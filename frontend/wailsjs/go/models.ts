export namespace main {
	
	export class SettingsDTO {
	    schedule_mode: string;
	    interval_hours: number;
	    allow_online_on_failure: boolean;
	    include_ipv6: boolean;
	    launch_at_login: boolean;
	    autostart_supported: boolean;
	    workers: number;
	    timeout_ms: number;
	    repeat: number;
	    domains: string[];
	
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
	        this.workers = source["workers"];
	        this.timeout_ms = source["timeout_ms"];
	        this.repeat = source["repeat"];
	        this.domains = source["domains"];
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
	    }
	}

}

