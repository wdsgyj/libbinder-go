package com.wdsgyj.libbinder.aidltest.javaclient;

import android.os.IBinder;
import com.wdsgyj.libbinder.aidltest.shared.BaselinePayload;
import com.wdsgyj.libbinder.aidltest.shared.IBaselineService;

public final class FixtureClientMain {
    public static final String DEFAULT_SERVICE_NAME = "libbinder.go.aidltest.baseline";

    private FixtureClientMain() {
    }

    public static void main(String[] args) throws Exception {
        String serviceName = args.length > 0 ? args[0] : DEFAULT_SERVICE_NAME;
        String expectedPrefix = args.length > 1 ? args[1] : "go";
        IBinder binder = FixtureServiceLookup.waitForService(serviceName, 5000);

        IBaselineService service = IBaselineService.Stub.asInterface(binder);
        boolean ping = service.Ping();
        String echo = service.EchoNullable("hello");
        BaselinePayload doubled = new BaselinePayload();
        BaselinePayload payload = new BaselinePayload();
        payload.code = 7;
        payload.note = "seed";
        int transform = service.Transform(11, doubled, payload);

        if (!ping) {
            throw new IllegalStateException("ping mismatch");
        }
        if (!java.util.Objects.equals(expectedPrefix + ":hello", echo)) {
            throw new IllegalStateException("echo mismatch: " + echo);
        }
        if (transform != 12) {
            throw new IllegalStateException("transform mismatch: " + transform);
        }
        if (doubled.code != 22 || !java.util.Objects.equals(expectedPrefix + ":doubled", doubled.note)) {
            throw new IllegalStateException("doubled mismatch");
        }
        if (payload.code != 29 || !java.util.Objects.equals(expectedPrefix + ":seed", payload.note)) {
            throw new IllegalStateException("payload mismatch");
        }

        System.out.println("OK");
    }
}
