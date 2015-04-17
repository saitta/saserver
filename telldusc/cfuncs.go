package telldus

//
// #include <stdio.h>
// #include <stdlib.h>
// #include <telldus-core.h>
//
//
// void sensorEvent_cgo(const char *protocol, const char *model, int sensorId, int dataType, const char *value, int ts, int callbackId, void *context) {
//   printf("C.sensorEvent(): called with arg = %d\n", sensorId);
//   sensorEvent_go(protocol, model, sensorId, dataType, value, ts, callbackId, context);
// }
//
// void deviceEvent_cgo(int deviceId, int method, const char *data, int callbackId, void *context) {
//   deviceEvent_go(deviceId, method, data, callbackId, context);
// }
import "C"
