package com.wdsgyj.libbinder.aidltest.javaserver;

import android.os.IBinder;
import java.lang.reflect.Method;

final class FixtureServiceRegistry {
    private FixtureServiceRegistry() {
    }

    static void addService(String name, IBinder binder) throws Exception {
        Class<?> cls = Class.forName("android.os.ServiceManager");
        Method method = cls.getDeclaredMethod("addService", String.class, IBinder.class);
        method.setAccessible(true);
        method.invoke(null, name, binder);
    }
}
