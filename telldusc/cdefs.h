#ifndef CDEFS_H
#define CDEFS_H

typedef void (*callback_fcn_sensor)(const char *, const char *, int , int , const char *, int , int , void *);
typedef void (*callback_fcn_device)(int , int , const char *, int , void *);

void sensorEvent_cgo(const char *protocol, const char *model, int sensorId, int dataType, const char *value, int ts, int callbackId, void *context);
void deviceEvent_cgo(int deviceId, int method, const char *data, int callbackId, void *context);
#endif