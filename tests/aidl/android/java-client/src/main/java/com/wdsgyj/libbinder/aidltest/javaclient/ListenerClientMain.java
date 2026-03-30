package com.wdsgyj.libbinder.aidltest.javaclient;

import android.os.IBinder;
import com.wdsgyj.libbinder.aidltest.shared.IListenerService;
import com.wdsgyj.libbinder.aidltest.shared.ListenerFixtures;

public final class ListenerClientMain {
    public static final String DEFAULT_SERVICE_NAME = "libbinder.go.aidltest.listener";

    private ListenerClientMain() {
    }

    public static void main(String[] args) throws Exception {
        String serviceName = args.length > 0 ? args[0] : DEFAULT_SERVICE_NAME;
        String mode = "basic";
        int rounds = 64;
        if (args.length > 1) {
            if ("basic".equals(args[1]) || "churn".equals(args[1])) {
                mode = args[1];
                if (args.length > 2) {
                    rounds = Integer.parseInt(args[2]);
                }
            } else if (args.length > 2) {
                mode = args[2];
                if (args.length > 3) {
                    rounds = Integer.parseInt(args[3]);
                }
            }
        }
        IBinder binder = FixtureServiceLookup.waitForService(serviceName, 5000);
        IListenerService service = IListenerService.Stub.asInterface(binder);
        if ("churn".equals(mode)) {
            ListenerFixtures.verifyChurn(service, rounds);
        } else {
            ListenerFixtures.verifyService(service);
        }
        System.out.println("OK");
    }
}
