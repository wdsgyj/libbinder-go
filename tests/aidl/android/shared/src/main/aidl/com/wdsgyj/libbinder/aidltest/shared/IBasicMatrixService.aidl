package com.wdsgyj.libbinder.aidltest.shared;

import com.wdsgyj.libbinder.aidltest.shared.BasicBundle;
import com.wdsgyj.libbinder.aidltest.shared.BasicEnvelope;
import com.wdsgyj.libbinder.aidltest.shared.BasicMode;
import com.wdsgyj.libbinder.aidltest.shared.BasicStringGroup;
import com.wdsgyj.libbinder.aidltest.shared.BasicUnion;
import com.wdsgyj.libbinder.aidltest.shared.BaselinePayload;

interface IBasicMatrixService {
  @nullable String EchoNullable(in @nullable String value);
  int[] ReverseInts(in int[] values);
  int[3] RotateTriple(in int[3] triple);
  List<String> DecorateTags(in List<String> tags);
  List<BasicStringGroup> DecorateTagGroups(in List<BasicStringGroup> groups);
  List<BaselinePayload> DecoratePayloads(in List<BaselinePayload> payloads);
  Map<String, String> DecorateLabels(in Map<String, String> labels);
  Map<String, BaselinePayload> DecoratePayloadMap(in Map<String, BaselinePayload> payloadMap);
  Map<String, List<BaselinePayload>> DecoratePayloadBuckets(in Map<String, List<BaselinePayload>> payloadBuckets);
  BasicMode FlipMode(in BasicMode mode);
  BasicUnion NormalizeUnion(in BasicUnion value);
  BasicBundle NormalizeBundle(in BasicBundle value);
  BasicEnvelope NormalizeEnvelope(in BasicEnvelope value);
  int ExpandBundle(in BasicBundle input, out BasicBundle doubled, inout BasicBundle payload);
}
