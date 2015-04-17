/** The xsan controller
 *
 * @author Leif Kalla
 */

var saserver = saserver || {};
var devices;


function main() {

	var url = "get_devices";


	$.getJSON(url, function(data, status) {
		if (status !== "success")
			throw "Cannot get devices";
        var keys = Object.keys(data);
        devices = {};
        for(var i=0,len = keys.length;i<len;i++) {
    		var dev = $.extend(new saserver.Device(), data[keys[i]]);
            devices[dev.Name] = dev;
        }

        console.log('devices:' + JSON.stringify(devices["dimmer_vrum"]) );

    });

}
