package com.wdsgyj.libbinder.aidltest.shared;

interface IAdvancedCallback {
  String OnSync(in String value);
  oneway void OnOneway(in String value);
}
