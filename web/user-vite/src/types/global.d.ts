declare global {
  interface Window {
    settings?: {
      base_url?: string;
      title?: string;
      version?: string;
      logo?: string;
      secure_path?: string;
      router_base?: string;
      disabled_modules?: string[];
    };
  }
}

export {};
