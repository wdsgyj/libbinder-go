package com.wdsgyj.libbinder.aidltest.shared;

import android.os.Binder;
import android.os.IBinder;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.ArrayBlockingQueue;
import java.util.concurrent.TimeUnit;

public final class ListenerFixtures {
    public static final String RAW_BINDER_DESCRIPTOR = "com.wdsgyj.libbinder.aidltest.shared.ListenerRawBinder";

    private ListenerFixtures() {
    }

    public static IBinder newRawBinder() {
        Binder binder = new Binder();
        binder.attachInterface(null, RAW_BINDER_DESCRIPTOR);
        return binder;
    }

    public static void verifyService(IListenerService service) throws Exception {
        IBinder echoed = service.EchoBinder(newRawBinder());
        assertTrue("EchoBinder.nonNil", echoed != null);
        assertEquals("EchoBinder.descriptor", RAW_BINDER_DESCRIPTOR, echoed.getInterfaceDescriptor());

        assertTrue("EchoBinder.nil", service.EchoBinder(null) == null);

        RecordingListener callback = new RecordingListener();
        service.RegisterListener(callback);
        service.RegisterListener(null);

        assertEquals("Emit.one", 1, service.Emit("one"));
        assertEquals("Callback.one", "one", callback.await(2000));

        assertEquals("Emit.two", 1, service.Emit("two"));
        assertEquals("Callback.two", "two", callback.await(2000));

        service.UnregisterListener(null);
        service.UnregisterListener(callback);

        assertEquals("Emit.three", 0, service.Emit("three"));
        if (callback.poll() != null) {
            throw new AssertionError("callback received event after unregister");
        }
    }

    public static void verifyChurn(IListenerService service, int rounds) throws Exception {
        RecordingListener callback = new RecordingListener();
        for (int i = 0; i < rounds; i++) {
            String value = String.format("churn-%03d", i);
            service.RegisterListener(callback);
            assertEquals("Emit." + i, 1, service.Emit(value));
            assertEquals("Callback." + i, value, callback.await(2000));
            service.UnregisterListener(callback);
            assertEquals("Emit.after." + i, 0, service.Emit(value + "-after"));
            if (callback.poll() != null) {
                throw new AssertionError("callback received event after unregister round " + i);
            }
        }
    }

    public static final class RecordingListener extends IListenerCallback.Stub {
        private final ArrayBlockingQueue<String> values = new ArrayBlockingQueue<>(8);

        @Override
        public void OnEvent(String value) {
            values.offer(value);
        }

        public String await(long timeoutMillis) throws Exception {
            String value = values.poll(timeoutMillis, TimeUnit.MILLISECONDS);
            if (value == null) {
                throw new IllegalStateException("listener callback timeout");
            }
            return value;
        }

        public String poll() {
            return values.poll();
        }
    }

    public static final class Registry {
        private final List<IListenerCallback> listeners = new ArrayList<>();

        public synchronized void register(IListenerCallback callback) {
            if (callback == null) {
                return;
            }
            listeners.add(callback);
        }

        public synchronized void unregister(IListenerCallback callback) {
            if (callback == null) {
                return;
            }
            IBinder target = callback.asBinder();
            listeners.removeIf(item -> item != null && item.asBinder() == target);
        }

        public int emit(String value) throws Exception {
            List<IListenerCallback> copy;
            synchronized (this) {
                copy = new ArrayList<>(listeners);
            }
            int count = 0;
            for (IListenerCallback callback : copy) {
                if (callback == null) {
                    continue;
                }
                callback.OnEvent(value);
                count++;
            }
            return count;
        }
    }

    private static void assertEquals(String label, String want, String got) {
        if (want == null ? got != null : !want.equals(got)) {
            throw new AssertionError(label + " = " + got + ", want " + want);
        }
    }

    private static void assertEquals(String label, int want, int got) {
        if (want != got) {
            throw new AssertionError(label + " = " + got + ", want " + want);
        }
    }

    private static void assertTrue(String label, boolean value) {
        if (!value) {
            throw new AssertionError(label);
        }
    }
}
