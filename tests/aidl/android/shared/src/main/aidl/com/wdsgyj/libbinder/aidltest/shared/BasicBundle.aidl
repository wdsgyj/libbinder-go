package com.wdsgyj.libbinder.aidltest.shared;

import com.wdsgyj.libbinder.aidltest.shared.BasicMode;
import com.wdsgyj.libbinder.aidltest.shared.BasicUnion;
import com.wdsgyj.libbinder.aidltest.shared.BaselinePayload;

parcelable BasicBundle {
  int[] ints;
  int[3] triple;
  @nullable String note;
  List<String> tags;
  List<BaselinePayload> payloads;
  Map<String, String> labels;
  Map<String, BaselinePayload> payloadMap;
  BasicMode mode = BasicMode.UNKNOWN;
  BasicUnion value;
}
