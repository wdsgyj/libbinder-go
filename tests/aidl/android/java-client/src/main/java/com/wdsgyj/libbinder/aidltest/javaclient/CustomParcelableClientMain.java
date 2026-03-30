package com.wdsgyj.libbinder.aidltest.javaclient;

import android.os.IBinder;
import com.wdsgyj.libbinder.aidltest.shared.CustomParcelableFixtures;
import com.wdsgyj.libbinder.aidltest.shared.ICustomParcelableService;

public final class CustomParcelableClientMain {
    public static final String DEFAULT_SERVICE_NAME = "libbinder.go.aidltest.customparcelable";

    private CustomParcelableClientMain() {
    }

    public static void main(String[] args) throws Exception {
        String serviceName = args.length > 0 ? args[0] : DEFAULT_SERVICE_NAME;
        String expectedPrefix = args.length > 1 ? args[1] : "go";
        IBinder binder = FixtureServiceLookup.waitForService(serviceName, 5000);
        CustomParcelableFixtures.verifyService(ICustomParcelableService.Stub.asInterface(binder), expectedPrefix);
        System.out.println("OK");
    }
}
