package com.wdsgyj.libbinder.aidltest.javaserver;

import android.os.Binder;

public final class FixtureServerMain {
    public static final String DEFAULT_SERVICE_NAME = "libbinder.go.aidltest.baseline";

    private FixtureServerMain() {
    }

    public static void main(String[] args) throws Exception {
        String serviceName = args.length > 0 ? args[0] : DEFAULT_SERVICE_NAME;
        FixtureServiceRegistry.addService(serviceName, new BaselineServiceImpl());
        System.out.println("registered " + serviceName);
        Binder.joinThreadPool();
    }
}
