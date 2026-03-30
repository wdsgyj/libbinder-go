package com.wdsgyj.libbinder.aidltest.shared;

import com.wdsgyj.libbinder.aidltest.shared.IAdvancedCallback;

interface IAdvancedService {
  IBinder EchoBinder(in IBinder input);
  String InvokeCallback(in IAdvancedCallback callback, in String value);
  oneway void FireOneway(in IAdvancedCallback callback, in String value);
  void FailServiceSpecific(in int code, in String message);
  String ReadFromFileDescriptor(in FileDescriptor fd);
  String ReadFromParcelFileDescriptor(in ParcelFileDescriptor fd);
}
