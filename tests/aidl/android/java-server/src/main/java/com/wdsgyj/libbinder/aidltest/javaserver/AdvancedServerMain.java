package com.wdsgyj.libbinder.aidltest.javaserver;

import android.os.Binder;
import com.wdsgyj.libbinder.aidltest.shared.AdvancedServiceProtocol;

public final class AdvancedServerMain {
    public static final String DEFAULT_SERVICE_NAME = "libbinder.go.aidltest.advanced";

    private AdvancedServerMain() {
    }

    public static void main(String[] args) throws Exception {
        String serviceName = args.length > 0 ? args[0] : DEFAULT_SERVICE_NAME;
        String prefix = args.length > 1 ? args[1] : "java";
        FixtureServiceRegistry.addService(serviceName, AdvancedServiceProtocol.newBinder(new AdvancedServiceImpl(prefix)));
        System.out.println("registered " + serviceName);
        Binder.joinThreadPool();
    }
}
