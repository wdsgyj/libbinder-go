package com.wdsgyj.libbinder.aidltest.javaclient;

import android.os.IBinder;
import com.wdsgyj.libbinder.aidltest.shared.AdvancedFixtures;
import com.wdsgyj.libbinder.aidltest.shared.AdvancedServiceProtocol;

public final class AdvancedClientMain {
    public static final String DEFAULT_SERVICE_NAME = "libbinder.go.aidltest.advanced";

    private AdvancedClientMain() {
    }

    public static void main(String[] args) throws Exception {
        String serviceName = args.length > 0 ? args[0] : DEFAULT_SERVICE_NAME;
        String expectedPrefix = args.length > 1 ? args[1] : "go";
        IBinder binder = FixtureServiceLookup.waitForService(serviceName, 5000);

        AdvancedServiceProtocol.Service service = AdvancedServiceProtocol.asInterface(binder);
        AdvancedFixtures.verifyService(service, expectedPrefix);
        System.out.println("OK");
    }
}
