export namespace main {
	
	export class SettingsDTO {
	    schedule_mode: string;
	    interval_hours: number;
	    allow_online_on_failure: boolean;
	    include_ipv6: boolean;
	    launch_at_login: boolean;
	    autostart_supported: boolean;
	    run_as_admin: boolean;
	    is_elevated: boolean;
	    needs_admin_relaunch: boolean;
	    elevation_label: string;
	    elevation_explain: string;
	    elevation_platform: string;
	    include_regional_pack: boolean;
	    workers: number;
	    timeout_ms: number;
	    repeat: number;
	    domains: string[];
	    regional_domains: string[];
	    effective_domains: string[];
	
	    static createFrom(source: any = {}) {
	        return new SettingsDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schedule_mode = source["schedule_mode"];
	        this.interval_hours = source["interval_hours"];
	        this.allow_online_on_failure = source["allow_online_on_failure"];
	        this.include_ipv6 = source["include_ipv6"];
	        this.launch_at_login = source["launch_at_login"];
	        this.autostart_supported = source["autostart_supported"];
	        this.run_as_admin = source["run_as_admin"];
	        this.is_elevated = source["is_elevated"];
	        this.needs_admin_relaunch = source["needs_admin_relaunch"];
	        this.elevation_label = source["elevation_label"];
	        this.elevation_explain = source["elevation_explain"];
	        this.elevation_platform = source["elevation_platform"];
	        this.include_regional_pack = source["include_regional_pack"];
	        this.workers = source["workers"];
	        this.timeout_ms = source["timeout_ms"];
	        this.repeat = source["repeat"];
	        this.domains = source["domains"];
	        this.regional_domains = source["regional_domains"];
	        this.effective_domains = source["effective_domains"];
	    }
	}
	export class StatusDTO {
	    last_check_at: string;
	    last_status: string;
	    last_message: string;
	    last_ip: string;
	    last_latency: string;
	    busy: boolean;
	    tray_line: string;
	    next_run_at: string;
	    proxy_bypass_at: string;
	    proxy_bypass_status: string;
	    proxy_bypass_message: string;
	    proxy_bypass_tooltip: string;
	
	    static createFrom(source: any = {}) {
	        return new StatusDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.last_check_at = source["last_check_at"];
	        this.last_status = source["last_status"];
	        this.last_message = source["last_message"];
	        this.last_ip = source["last_ip"];
	        this.last_latency = source["last_latency"];
	        this.busy = source["busy"];
	        this.tray_line = source["tray_line"];
	        this.next_run_at = source["next_run_at"];
	        this.proxy_bypass_at = source["proxy_bypass_at"];
	        this.proxy_bypass_status = source["proxy_bypass_status"];
	        this.proxy_bypass_message = source["proxy_bypass_message"];
	        this.proxy_bypass_tooltip = source["proxy_bypass_tooltip"];
	    }
	}

}

