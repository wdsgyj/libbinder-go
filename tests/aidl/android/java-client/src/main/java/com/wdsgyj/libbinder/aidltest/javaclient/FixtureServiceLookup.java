package com.wdsgyj.libbinder.aidltest.javaclient;

import android.os.IBinder;
import java.lang.reflect.Method;

final class FixtureServiceLookup {
    private FixtureServiceLookup() {
    }

    static IBinder checkService(String name) throws Exception {
        Class<?> cls = Class.forName("android.os.ServiceManager");
        Method method = cls.getDeclaredMethod("checkService", String.class);
        method.setAccessible(true);
        return (IBinder) method.invoke(null, name);
    }

    static IBinder waitForService(String name, long timeoutMillis) throws Exception {
        long deadline = System.currentTimeMillis() + timeoutMillis;
        while (true) {
            IBinder binder = checkService(name);
            if (binder != null) {
                return binder;
            }
            if (System.currentTimeMillis() >= deadline) {
                throw new IllegalStateException("service not found: " + name);
            }
            Thread.sleep(100);
        }
    }

    static String[] listServices() throws Exception {
        Class<?> cls = Class.forName("android.os.ServiceManager");
        try {
            Method method = cls.getDeclaredMethod("listServices");
            method.setAccessible(true);
            return (String[]) method.invoke(null);
        } catch (NoSuchMethodException ignored) {
            Method method = cls.getDeclaredMethod("listServices", int.class);
            method.setAccessible(true);
            return (String[]) method.invoke(null, 15);
        }
    }
}
