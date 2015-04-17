/** The xsan controller
 *
 * @author Leif Kalla
 */

var saserver = saserver || {};

/**
 * Command parameter , can be global or detail
 *  @constructor
 */
saserver.Device = function() {
	this.Name  = "None";
};

saserver.Device.prototype.MytoString = function() {
	out = "name:" + this.Name + " " + this.Id + " " + this.Type;

    //for(var i=0,len = this.details.length;i<len;i++) {
	//    out += '\n\t' + this.details[i].toString();
	//}
	return out;
};


function getRandomInt(min, max) {
	return Math.floor(Math.random() * (max - min + 1) + min);
};


function clone(obj){
    if(obj == null || typeof(obj) != 'object')
        return obj;

    var temp = obj.constructor(); // changed

    for(var key in obj)
        temp[key] = clone(obj[key]);
    return temp;
}

function parseHexStr(mystr) {
	
    var arr = new Array((mystr.length-2)/2);
	
	if(mystr.length>1) {
	    if(mystr.substr(0,2) === '0x') {
	        for(var i=2, j=0, len = mystr.length; i<len;i+=2) {
	            arr[j++]=parseInt(mystr.substr(i,2),16);
	        }
	    }
	}
	return arr;	
}

