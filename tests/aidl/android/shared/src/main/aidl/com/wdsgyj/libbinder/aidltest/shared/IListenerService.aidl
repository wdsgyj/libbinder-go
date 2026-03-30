package com.wdsgyj.libbinder.aidltest.shared;

import com.wdsgyj.libbinder.aidltest.shared.IListenerCallback;

interface IListenerService {
  void RegisterListener(in IListenerCallback callback);
  void UnregisterListener(in IListenerCallback callback);
  int Emit(in String value);
  IBinder EchoBinder(in IBinder input);
}
