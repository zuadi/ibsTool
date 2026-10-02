export namespace udpdiscovery {
	
	export class Device {
	    hostname: string;
	    ip: string;
	    port: string;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new Device(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hostname = source["hostname"];
	        this.ip = source["ip"];
	        this.port = source["port"];
	        this.url = source["url"];
	    }
	}

}

