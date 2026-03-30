package com.wdsgyj.libbinder.aidltest.javaclient;

import android.os.IBinder;
import com.wdsgyj.libbinder.aidltest.shared.IBaselineService;
import java.util.Arrays;
import java.util.Objects;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;

public final class LifecycleClientMain {
    public static final String DEFAULT_SERVICE_NAME = "libbinder.go.aidltest.baseline";

    private LifecycleClientMain() {
    }

    public static void main(String[] args) throws Exception {
        String mode = args.length > 0 ? args[0] : "discovery";
        String serviceName = args.length > 1 ? args[1] : DEFAULT_SERVICE_NAME;
        String expectedPrefix = args.length > 2 ? args[2] : "go";
        int killPid = args.length > 3 ? Integer.parseInt(args[3]) : 0;
        long killDelayMillis = args.length > 4 ? Long.parseLong(args[4]) : 500;

        switch (mode) {
            case "discovery":
                verifyDiscovery(serviceName, expectedPrefix);
                break;
            case "death":
                verifyDeath(serviceName, killPid, killDelayMillis);
                break;
            default:
                throw new IllegalArgumentException("unsupported mode: " + mode);
        }

        System.out.println("OK");
    }

    private static void verifyDiscovery(String serviceName, String expectedPrefix) throws Exception {
        IBinder binder = FixtureServiceLookup.waitForService(serviceName, 5000);
        if (binder == null) {
            throw new IllegalStateException("waitForService returned null");
        }
        if (FixtureServiceLookup.checkService(serviceName) == null) {
            throw new IllegalStateException("checkService returned null");
        }
        if (!Arrays.asList(FixtureServiceLookup.listServices()).contains(serviceName)) {
            throw new IllegalStateException("listServices missing " + serviceName);
        }

        IBaselineService service = IBaselineService.Stub.asInterface(binder);
        if (!service.Ping()) {
            throw new IllegalStateException("ping mismatch");
        }
        if (!Objects.equals(expectedPrefix + ":hello", service.EchoNullable("hello"))) {
            throw new IllegalStateException("echo mismatch");
        }
    }

    private static void verifyDeath(String serviceName, int killPid, long killDelayMillis) throws Exception {
        if (killPid <= 0) {
            throw new IllegalArgumentException("death mode requires kill pid");
        }
        IBinder binder = FixtureServiceLookup.waitForService(serviceName, 5000);
        CountDownLatch latch = new CountDownLatch(1);
        binder.linkToDeath(latch::countDown, 0);

        Thread killer = new Thread(() -> {
            try {
                Thread.sleep(killDelayMillis);
                Runtime.getRuntime().exec(new String[] {"sh", "-c", "kill -9 " + killPid}).waitFor();
            } catch (Exception e) {
                throw new RuntimeException(e);
            }
        }, "lifecycle-killer");
        killer.start();

        if (!latch.await(10, TimeUnit.SECONDS)) {
            throw new IllegalStateException("death recipient timeout");
        }
    }
}
