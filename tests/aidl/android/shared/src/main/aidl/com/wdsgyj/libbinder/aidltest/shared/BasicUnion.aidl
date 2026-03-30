package com.wdsgyj.libbinder.aidltest.shared;

import com.wdsgyj.libbinder.aidltest.shared.BaselinePayload;

union BasicUnion {
  int number;
  @nullable String text;
  BaselinePayload payload;
}
