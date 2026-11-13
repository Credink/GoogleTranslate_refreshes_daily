export function GetSettings(): Promise<SettingsDTO>;
export function GetStatus(): Promise<StatusDTO>;
export function Quit(): Promise<void>;
export function RefreshNow(): Promise<StatusDTO>;
export function SaveSettings(arg1: SettingsDTO): Promise<string>;
export function TrayMenuLabels(): Promise<Record<string, string>>;

export interface StatusDTO {
  last_check_at: string;
  last_status: string;
  last_message: string;
  last_ip: string;
  last_latency: string;
  busy: boolean;
  tray_line: string;
  next_run_at: string;
}

export interface SettingsDTO {
  schedule_mode: string;
  interval_hours: number;
  online_sync: boolean;
  include_ipv6: boolean;
  workers: number;
  timeout_ms: number;
  repeat: number;
  domains: string[];
}
