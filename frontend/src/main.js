import './style.css';
import './app.css';

import {
    GetStatus,
    GetSettings,
    SaveSettings,
    RefreshNow,
    TrayMenuLabels,
} from '../wailsjs/go/main/App';

const app = document.querySelector('#app');

app.innerHTML = `
  <main class="shell">
    <header>
      <h1>谷歌翻译 Hosts 更新器</h1>
      <p class="sub">按计划检测 translate.googleapis.com；仅在当前 IP 失效时拉取远程 IP 库并更新 hosts。</p>
    </header>

    <section class="panel">
      <h2>状态</h2>
      <dl id="status">
        <div><dt>上次检查</dt><dd id="lastCheck">—</dd></div>
        <div><dt>结果</dt><dd id="lastStatus">—</dd></div>
        <div><dt>说明</dt><dd id="lastMessage">—</dd></div>
        <div><dt>当前/最佳 IP</dt><dd id="lastIP">—</dd></div>
        <div><dt>延迟</dt><dd id="lastLatency">—</dd></div>
        <div><dt>下次计划</dt><dd id="nextRun">—</dd></div>
        <div><dt>托盘摘要</dt><dd id="trayLine">—</dd></div>
      </dl>
      <div class="actions">
        <button id="btnRefresh" class="btn primary" type="button">立即刷新</button>
        <button id="btnReload" class="btn" type="button">刷新状态</button>
      </div>
      <p id="busyNote" class="note" hidden>正在检查中…</p>
    </section>

    <section class="panel">
      <h2>设置</h2>
      <label class="field">
        <span>检查频率</span>
        <select id="scheduleMode">
          <option value="daily_midnight">每天本地午夜一次（默认）</option>
          <option value="interval_hours">按小时间隔</option>
        </select>
      </label>
      <label class="field">
        <span>间隔小时数</span>
        <input id="intervalHours" type="number" min="1" max="168" value="24" />
      </label>
      <label class="check">
        <input id="allowOnline" type="checkbox" checked />
        <span>当前 IP 失效时，允许在线拉取 IP 库（不定时同步）</span>
      </label>
      <label class="check">
        <input id="includeIPv6" type="checkbox" />
        <span>启用 IPv6（默认关；不可用时本轮自动回退 IPv4，双栈优先 IPv4）</span>
      </label>
      <div class="actions">
        <button id="btnSave" class="btn primary" type="button">保存设置</button>
      </div>
      <p id="saveNote" class="note"></p>
    </section>

    <footer>
      <p id="trayHint" class="foot"></p>
    </footer>
  </main>
`;

function fillStatus(s) {
    document.getElementById('lastCheck').textContent = s.last_check_at || '尚未检查';
    document.getElementById('lastStatus').textContent = s.last_status || '—';
    document.getElementById('lastMessage').textContent = s.last_message || '—';
    document.getElementById('lastIP').textContent = s.last_ip || '—';
    document.getElementById('lastLatency').textContent = s.last_latency || '—';
    document.getElementById('nextRun').textContent = s.next_run_at || '—';
    document.getElementById('trayLine').textContent = s.tray_line || '—';
    document.getElementById('busyNote').hidden = !s.busy;
    document.getElementById('btnRefresh').disabled = !!s.busy;
}

function fillSettings(s) {
    document.getElementById('scheduleMode').value = s.schedule_mode || 'daily_midnight';
    document.getElementById('intervalHours').value = s.interval_hours || 24;
    document.getElementById('allowOnline').checked = s.allow_online_on_failure !== false;
    document.getElementById('includeIPv6').checked = !!s.include_ipv6;
}

async function reload() {
    const [status, settings, tray] = await Promise.all([
        GetStatus(),
        GetSettings(),
        TrayMenuLabels(),
    ]);
    fillStatus(status);
    fillSettings(settings);
    document.getElementById('trayHint').textContent =
        `托盘：${tray.refresh} / ${tray.settings} / ${tray.quit}。${tray.note || ''}`;
}

document.getElementById('btnReload').addEventListener('click', () => {
    reload().catch(console.error);
});

document.getElementById('btnRefresh').addEventListener('click', async () => {
    document.getElementById('busyNote').hidden = false;
    document.getElementById('btnRefresh').disabled = true;
    try {
        const s = await RefreshNow();
        fillStatus(s);
    } catch (e) {
        console.error(e);
        document.getElementById('lastMessage').textContent = String(e);
    } finally {
        await reload();
    }
});

document.getElementById('btnSave').addEventListener('click', async () => {
    const payload = {
        schedule_mode: document.getElementById('scheduleMode').value,
        interval_hours: Number(document.getElementById('intervalHours').value) || 24,
        allow_online_on_failure: document.getElementById('allowOnline').checked,
        include_ipv6: document.getElementById('includeIPv6').checked,
        workers: 64,
        timeout_ms: 1500,
        repeat: 3,
        domains: ['translate.googleapis.com', 'translate-pa.googleapis.com'],
    };
    const msg = await SaveSettings(payload);
    document.getElementById('saveNote').textContent = msg;
    await reload();
});

reload().catch(console.error);
setInterval(() => {
    GetStatus().then(fillStatus).catch(() => {});
}, 5000);
