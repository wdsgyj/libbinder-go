package com.wdsgyj.libbinder.aidltest.shared;

import com.wdsgyj.libbinder.aidltest.shared.BasicBundle;
import com.wdsgyj.libbinder.aidltest.shared.BaselinePayload;

parcelable BasicEnvelope {
  String title = "untitled";
  @nullable String note;
  @nullable BaselinePayload primary;
  List<BaselinePayload> history;
  BasicBundle bundle;
}
