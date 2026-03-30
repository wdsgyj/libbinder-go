package com.wdsgyj.libbinder.aidltest.shared;

import com.wdsgyj.libbinder.aidltest.shared.CustomBox;

interface ICustomParcelableService {
  CustomBox Normalize(in CustomBox value);
  @nullable CustomBox NormalizeNullable(in @nullable CustomBox value);
}
