import './style.css';
import './app.css';

import {
    GetStatus,
    GetSettings,
    SaveSettings,
    RefreshNow,
    RelaunchAsAdmin,
    ConfigureProxyBypass,
    TrayMenuLabels,
    GetBuildInfo,
} from '../wailsjs/go/main/App';

const app = document.querySelector('#app');

app.innerHTML = `
  <main class="shell">
    <header>
      <h1>谷歌翻译 Hosts 更新器</h1>
      <p class="sub">按计划探测 translate.googleapis.com；优选 IP 写入默认托管域名（含网页入口与 Chrome 内置翻译相关主机）；仅在当前 IP 失效时拉取远程 IP 库。</p>
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
        <div><dt>权限模式</dt><dd id="elevationStatus">—</dd></div>
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
          <option value="on_app_start">软件启动时刷新（默认）</option>
          <option value="daily_midnight">每天本地午夜一次</option>
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
      <label class="check" id="autostartRow">
        <input id="launchAtLogin" type="checkbox" />
        <span>开机自启动（登录时启动：Windows Run / macOS LaunchAgent / Linux XDG Autostart）</span>
      </label>

      <div class="domains-block">
        <div class="section-title-row">
          <h3 class="section-title">Hosts / 代理相关</h3>
          <button
            id="proxyHintIcon"
            class="hint-icon"
            type="button"
            aria-label="代理插件提示"
            title=""
          >
            <span class="hint-icon-ring" aria-hidden="true">!</span>
          </button>
        </div>
        <p id="proxyHintTooltip" class="note proxy-hint" hidden></p>
        <label class="field">
          <span>托管域名（每行一个；同一优选 IP）</span>
          <textarea id="domainsList" rows="6" spellcheck="false"></textarea>
        </label>
        <label class="check">
          <input id="includeRegional" type="checkbox" />
          <span>额外写入地区前端（ccTLD，默认关）</span>
        </label>
        <p id="regionalHint" class="note"></p>
        <p id="effectiveHint" class="note"></p>
        <p id="proxyBypassStatus" class="note"></p>
        <div class="actions">
          <button id="btnProxyBypass" class="btn" type="button">配置代理插件绕过</button>
        </div>
      </div>

      <div class="elevation-block">
        <label class="check">
          <input id="runAsAdmin" type="checkbox" />
          <span>以管理员身份运行（可选）</span>
        </label>
        <p id="elevationExplain" class="note elevation-explain"></p>
        <p id="elevationLabel" class="note"></p>
        <p id="elevationPlatform" class="note"></p>
        <div class="actions" id="adminRelaunchRow" hidden>
          <button id="btnRelaunchAdmin" class="btn" type="button">立即以管理员身份重启</button>
        </div>
      </div>

      <div class="actions">
        <button id="btnSave" class="btn primary" type="button">保存设置</button>
      </div>
      <p id="saveNote" class="note"></p>
    </section>

    <footer>
      <p id="trayHint" class="foot"></p>
      <p class="foot">写入 hosts 后会刷新系统 DNS 缓存。若浏览器仍不通：关闭 Chrome/Edge「使用安全 DNS」(DoH，会绕过 hosts)。探测始终针对 <code>translate.googleapis.com</code>；默认还会写入 <code>jnn-pa.googleapis.com</code>、<code>translate.google.com</code>、<code>translate.google.com.hk</code> 等。请收藏 <code>translate.google.com</code> / <code>.com.hk</code>；勿依赖 <code>translate.google.cn</code>（仅跳板）。</p>
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

    const tip =
        s.proxy_bypass_tooltip ||
        '本软件已尝试自动配置 ZeroOmega 等 Chrome 代理插件，将托管的翻译域名设为直连（DIRECT/绕过代理）。若翻译仍失败，请在代理插件中手动把这些域名设为 bypass/DIRECT。本功能无法覆盖全部代理扩展。';
    const icon = document.getElementById('proxyHintIcon');
    icon.title = tip;
    icon.setAttribute('aria-label', tip);
    document.getElementById('proxyHintTooltip').textContent = tip;

    const bypassBits = [];
    if (s.proxy_bypass_status) bypassBits.push(`状态: ${s.proxy_bypass_status}`);
    if (s.proxy_bypass_at) bypassBits.push(`时间: ${s.proxy_bypass_at}`);
    if (s.proxy_bypass_message) bypassBits.push(s.proxy_bypass_message);
    document.getElementById('proxyBypassStatus').textContent = bypassBits.length
        ? `代理插件自动配置 — ${bypassBits.join('；')}`
        : '代理插件自动配置 — 尚未执行（写入 hosts 后会尝试；也可点下方按钮）';
}

function fillSettings(s) {
    document.getElementById('scheduleMode').value = s.schedule_mode || 'on_app_start';
    document.getElementById('intervalHours').value = s.interval_hours || 24;
    document.getElementById('allowOnline').checked = s.allow_online_on_failure !== false;
    document.getElementById('includeIPv6').checked = !!s.include_ipv6;
    document.getElementById('launchAtLogin').checked = !!s.launch_at_login;
    document.getElementById('runAsAdmin').checked = !!s.run_as_admin;
    document.getElementById('includeRegional').checked = !!s.include_regional_pack;
    const domains = Array.isArray(s.domains) ? s.domains : [];
    document.getElementById('domainsList').value = domains.join('\n');
    const regional = Array.isArray(s.regional_domains) ? s.regional_domains : [];
    document.getElementById('regionalHint').textContent = regional.length
        ? `地区包：${regional.join('、')}`
        : '';
    const effective = Array.isArray(s.effective_domains) ? s.effective_domains : domains;
    document.getElementById('effectiveHint').textContent =
        `实际写入 ${effective.length} 个域名（探测仍仅 translate.googleapis.com）。`;
    document.getElementById('elevationExplain').textContent =
        s.elevation_explain ||
        '管理员模式可长期静默更新 hosts（换 IP 时不再每次点授权）。普通模式更安全，但每次真正改 hosts 都需要点击授予权限。';
    document.getElementById('elevationLabel').textContent = s.elevation_label || '';
    document.getElementById('elevationPlatform').textContent = s.elevation_platform || '';
    document.getElementById('elevationStatus').textContent =
        (s.elevation_label || '—') + (s.run_as_admin ? '（偏好：管理员）' : '（偏好：普通用户）');
    const row = document.getElementById('autostartRow');
    if (s.autostart_supported === false) {
        row.hidden = true;
    } else {
        row.hidden = false;
    }
    document.getElementById('adminRelaunchRow').hidden = !s.needs_admin_relaunch;
}

function parseDomains(text) {
    return String(text || '')
        .split(/[\n,;]+/)
        .map((s) => s.trim())
        .filter(Boolean);
}

async function reload() {
    const [status, settings, tray, build] = await Promise.all([
        GetStatus(),
        GetSettings(),
        TrayMenuLabels(),
        GetBuildInfo().catch(() => ''),
    ]);
    fillStatus(status);
    fillSettings(settings);
    const buildNote = build ? ` 构建：${build}` : '';
    document.getElementById('trayHint').textContent =
        `托盘：${tray.show || '显示窗口'} / ${tray.refresh} / ${tray.settings} / ${tray.quit}。${tray.note || ''}${buildNote}`;
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
    const wantAdmin = document.getElementById('runAsAdmin').checked;
    document.getElementById('saveNote').textContent = wantAdmin
        ? '保存中…若需提权请在系统授权提示中确认。'
        : '保存中…';
    const payload = {
        schedule_mode: document.getElementById('scheduleMode').value,
        interval_hours: Number(document.getElementById('intervalHours').value) || 24,
        allow_online_on_failure: document.getElementById('allowOnline').checked,
        include_ipv6: document.getElementById('includeIPv6').checked,
        launch_at_login: document.getElementById('launchAtLogin').checked,
        run_as_admin: wantAdmin,
        include_regional_pack: document.getElementById('includeRegional').checked,
        workers: 32,
        timeout_ms: 800,
        repeat: 2,
        domains: parseDomains(document.getElementById('domainsList').value),
    };
    try {
        const msg = await SaveSettings(payload);
        document.getElementById('saveNote').textContent = msg;
    } catch (e) {
        document.getElementById('saveNote').textContent = String(e);
    }
    await reload();
});

document.getElementById('btnRelaunchAdmin').addEventListener('click', async () => {
    document.getElementById('saveNote').textContent = '请在系统授权提示中确认…';
    try {
        const msg = await RelaunchAsAdmin();
        document.getElementById('saveNote').textContent = msg;
    } catch (e) {
        document.getElementById('saveNote').textContent = String(e);
    }
    await reload();
});

const proxyHintIcon = document.getElementById('proxyHintIcon');
const proxyHintTooltip = document.getElementById('proxyHintTooltip');
proxyHintIcon.addEventListener('mouseenter', () => {
    proxyHintTooltip.hidden = false;
});
proxyHintIcon.addEventListener('mouseleave', () => {
    proxyHintTooltip.hidden = true;
});
proxyHintIcon.addEventListener('focus', () => {
    proxyHintTooltip.hidden = false;
});
proxyHintIcon.addEventListener('blur', () => {
    proxyHintTooltip.hidden = true;
});

document.getElementById('btnProxyBypass').addEventListener('click', async () => {
    document.getElementById('proxyBypassStatus').textContent = '正在配置代理插件绕过…（若浏览器正在运行可能失败）';
    try {
        const s = await ConfigureProxyBypass();
        fillStatus(s);
    } catch (e) {
        document.getElementById('proxyBypassStatus').textContent = String(e);
    }
});

reload().catch(console.error);
setInterval(() => {
    GetStatus().then(fillStatus).catch(() => {});
}, 5000);
