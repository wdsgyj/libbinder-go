package com.wdsgyj.libbinder.aidltest.javaserver;

import com.wdsgyj.libbinder.aidltest.shared.BaselinePayload;
import com.wdsgyj.libbinder.aidltest.shared.IBaselineService;

public final class BaselineServiceImpl extends IBaselineService.Stub {
    @Override
    public boolean Ping() {
        return true;
    }

    @Override
    public String EchoNullable(String value) {
        if (value == null) {
            return null;
        }
        return "java:" + value;
    }

    @Override
    public int Transform(int input, BaselinePayload doubled, BaselinePayload payload) {
        int value = input * 2;
        if (doubled != null) {
            doubled.code = value;
            doubled.note = "java:doubled";
        }
        if (payload != null) {
            payload.code += value;
            payload.note = payload.note == null ? "java:default" : "java:" + payload.note;
        }
        return input + 1;
    }
}
