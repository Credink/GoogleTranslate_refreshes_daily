package chromeproxy

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// patchStorageDir checks Omega profiles first; writes LevelDB only when a real
// fix is needed (missing bypass/DIRECT or typo). Already-correct DBs are never rewritten.
func patchStorageDir(dir string, domains []string, dryRun bool) (ProfilePatchResult, error) {
	res := ProfilePatchResult{StorageDir: dir}

	pending, schema, notes, err := planPatches(dir, domains)
	if err != nil {
		return res, err
	}
	res.SchemaVersion = schema
	res.Details = notes

	if schema != "" && schema != "1" && schema != "2" {
		res.Skipped = true
		res.Message = "未知 schemaVersion=" + schema + "，已跳过以免损坏配置"
		return res, nil
	}

	if len(pending) == 0 {
		res.Message = "已检查：托管域名已在 bypass/DIRECT，跳过写入"
		return res, nil
	}

	if dryRun {
		res.Changed = true
		res.Message = fmt.Sprintf("预览：将更新 %d 个配置项（当前缺失或有笔误）", len(pending))
		return res, nil
	}

	db, err := leveldb.OpenFile(dir, &opt.Options{})
	if err != nil {
		if dbRO, errRO := leveldb.OpenFile(dir, &opt.Options{ReadOnly: true}); errRO == nil {
			_ = dbRO.Close()
			return res, fmt.Errorf("无法写入扩展存储（浏览器可能正在运行，请完全退出 Chrome/Edge 后重试）: %w", err)
		}
		return res, fmt.Errorf("打开 LevelDB 失败: %w", err)
	}
	defer db.Close()

	batch := new(leveldb.Batch)
	for _, w := range pending {
		batch.Put([]byte(w.key), w.value)
	}
	if err := db.Write(batch, nil); err != nil {
		return res, fmt.Errorf("写入 LevelDB 失败: %w", err)
	}
	res.Changed = true
	res.Details = notes
	res.Message = fmt.Sprintf("已更新 %d 个配置项；请完全重启浏览器使代理插件重新加载", len(pending))
	return res, nil
}

type pendingWrite struct {
	key   string
	value []byte
}

// planPatches opens the DB read-only and returns only keys that actually need changes.
func planPatches(dir string, domains []string) (writes []pendingWrite, schema string, notes []string, err error) {
	db, err := leveldb.OpenFile(dir, &opt.Options{ReadOnly: true})
	if err != nil {
		return nil, "", nil, fmt.Errorf("打开 LevelDB 失败: %w", err)
	}
	defer db.Close()

	if raw, getErr := db.Get([]byte("schemaVersion"), nil); getErr == nil {
		schema = strings.TrimSpace(string(raw))
	}

	iter := db.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		key := string(iter.Key())
		if !strings.HasPrefix(key, "+") {
			continue
		}
		val := append([]byte(nil), iter.Value()...)
		if !json.Valid(val) || !strings.Contains(string(val), "profileType") {
			continue
		}
		out, changed, detail, patchErr := PatchProfileJSON(val, domains)
		if patchErr != nil {
			notes = append(notes, fmt.Sprintf("%s: 解析失败 (%v)", key, patchErr))
			continue
		}
		if !changed {
			continue
		}
		// Defense: never queue a write if marshaled payload equals existing bytes.
		if string(out) == string(val) {
			continue
		}
		notes = append(notes, fmt.Sprintf("%s: %s", key, detail))
		writes = append(writes, pendingWrite{key: key, value: out})
	}
	if err := iter.Error(); err != nil {
		return nil, schema, notes, fmt.Errorf("遍历 LevelDB: %w", err)
	}
	return writes, schema, notes, nil
}

// ProfilePatchResult is the outcome for one extension storage DB.
type ProfilePatchResult struct {
	StorageDir    string
	SchemaVersion string
	Changed       bool
	Skipped       bool
	Message       string
	Details       []string
}
