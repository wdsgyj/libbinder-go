package com.wdsgyj.libbinder.aidltest.javaclient;

import android.os.IBinder;
import com.wdsgyj.libbinder.aidltest.shared.BasicMatrixFixtures;
import com.wdsgyj.libbinder.aidltest.shared.IBasicMatrixService;

public final class BasicMatrixClientMain {
    public static final String DEFAULT_SERVICE_NAME = "libbinder.go.aidltest.matrix";

    private BasicMatrixClientMain() {
    }

    public static void main(String[] args) throws Exception {
        String serviceName = args.length > 0 ? args[0] : DEFAULT_SERVICE_NAME;
        String expectedPrefix = args.length > 1 ? args[1] : "go";
        IBinder binder = FixtureServiceLookup.waitForService(serviceName, 5000);

        IBasicMatrixService service = IBasicMatrixService.Stub.asInterface(binder);
        BasicMatrixFixtures.verifyService(service, expectedPrefix);
        System.out.println("OK");
    }
}
