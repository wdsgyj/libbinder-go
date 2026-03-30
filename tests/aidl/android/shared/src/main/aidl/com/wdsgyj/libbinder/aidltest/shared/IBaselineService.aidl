package com.wdsgyj.libbinder.aidltest.shared;

import com.wdsgyj.libbinder.aidltest.shared.BaselinePayload;

interface IBaselineService {
  boolean Ping();
  @nullable String EchoNullable(@nullable String value);
  int Transform(in int input, out BaselinePayload doubled, inout BaselinePayload payload);
}
