export namespace config {
	
	export class ImmichConfig {
	    serverURL: string;
	    apiKey: string;
	
	    static createFrom(source: any = {}) {
	        return new ImmichConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serverURL = source["serverURL"];
	        this.apiKey = source["apiKey"];
	    }
	}
	export class Config {
	    immich: ImmichConfig;
	    targetFolders: string[];
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.immich = this.convertValues(source["immich"], ImmichConfig);
	        this.targetFolders = source["targetFolders"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

