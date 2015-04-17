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

        var junk = {"nisse":1,"valter":2};
        console.log(JSON.stringify(junk));
        console.log('devices:' + JSON.stringify(devices["dimmer_vrum"]) );

        /*
        var keys = Object.keys(devices);
        for (var i=0,len = keys.length;i<len;i++) {
            $("#devices").append(devices[keys[i]].Name + "<br>");
        } */

    });

}

