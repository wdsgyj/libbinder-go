package com.wdsgyj.libbinder.aidltest.shared;

import android.os.Binder;
import android.os.IBinder;
import android.os.ParcelFileDescriptor;
import android.os.RemoteException;
import java.io.File;
import java.io.FileDescriptor;
import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.IOException;
import java.lang.reflect.Field;
import java.nio.charset.StandardCharsets;
import java.util.concurrent.ArrayBlockingQueue;
import java.util.concurrent.TimeUnit;

public final class AdvancedFixtures {
    public static final String RAW_BINDER_DESCRIPTOR = "com.wdsgyj.libbinder.aidltest.shared.RawBinder";

    private AdvancedFixtures() {
    }

    public static IBinder echoBinder(IBinder input) {
        return input;
    }

    public static String invokeCallback(String prefix, IAdvancedCallback callback, String value) throws RemoteException {
        return prefix + ":" + callback.OnSync(value);
    }

    public static void fireOneway(String prefix, IAdvancedCallback callback, String value) throws RemoteException {
        callback.OnOneway(prefix + ":" + value);
    }

    public static String readAll(FileDescriptor fd) throws IOException {
        try (FileInputStream in = new FileInputStream(fd)) {
            return new String(in.readAllBytes(), StandardCharsets.UTF_8);
        }
    }

    public static String readAll(ParcelFileDescriptor fd) throws IOException {
        try (ParcelFileDescriptor.AutoCloseInputStream in = new ParcelFileDescriptor.AutoCloseInputStream(fd)) {
            return new String(in.readAllBytes(), StandardCharsets.UTF_8);
        }
    }

    public static IBinder newRawBinder() {
        Binder binder = new Binder();
        binder.attachInterface(null, RAW_BINDER_DESCRIPTOR);
        return binder;
    }

    public static void verifyService(AdvancedServiceProtocol.Service service, String servicePrefix) throws Exception {
        IBinder echoed = service.EchoBinder(newRawBinder());
        assertTrue("EchoBinder.null", echoed != null);
        assertEquals("EchoBinder.descriptor", RAW_BINDER_DESCRIPTOR, echoed.getInterfaceDescriptor());

        RecordingCallback callback = new RecordingCallback("java-callback");
        String reply = service.InvokeCallback(callback, "sync-value");
        assertEquals("InvokeCallback.reply", servicePrefix + ":java-callback:sync-value", reply);
        assertEquals("InvokeCallback.arg", "sync-value", callback.awaitSync(2000));

        service.FireOneway(callback, "oneway-value");
        assertEquals("FireOneway.arg", servicePrefix + ":oneway-value", callback.awaitOneway(2000));

        try {
            service.FailServiceSpecific(27, "boom");
            throw new AssertionError("FailServiceSpecific did not throw");
        } catch (RuntimeException expected) {
            assertServiceSpecificException(expected, 27, "boom");
        }

        File fdFile = writeTempFile("advanced-fd-", "fd-payload");
        try (ParcelFileDescriptor fd = ParcelFileDescriptor.open(fdFile, ParcelFileDescriptor.MODE_READ_ONLY)) {
            assertEquals("ReadFromFileDescriptor", "fd-payload", service.ReadFromFileDescriptor(fd.getFileDescriptor()));
        } finally {
            //noinspection ResultOfMethodCallIgnored
            fdFile.delete();
        }

        File pfdFile = writeTempFile("advanced-pfd-", "pfd-payload");
        try (ParcelFileDescriptor fd = ParcelFileDescriptor.open(pfdFile, ParcelFileDescriptor.MODE_READ_ONLY)) {
            assertEquals("ReadFromParcelFileDescriptor", "pfd-payload", service.ReadFromParcelFileDescriptor(fd));
        } finally {
            //noinspection ResultOfMethodCallIgnored
            pfdFile.delete();
        }
    }

    public static final class RecordingCallback extends IAdvancedCallback.Stub {
        private final String prefix;
        private final ArrayBlockingQueue<String> syncValues = new ArrayBlockingQueue<>(4);
        private final ArrayBlockingQueue<String> onewayValues = new ArrayBlockingQueue<>(4);

        public RecordingCallback(String prefix) {
            this.prefix = prefix;
        }

        @Override
        public String OnSync(String value) {
            syncValues.offer(value);
            return prefix + ":" + value;
        }

        @Override
        public void OnOneway(String value) {
            onewayValues.offer(value);
        }

        public String awaitSync(long timeoutMillis) throws Exception {
            String value = syncValues.poll(timeoutMillis, TimeUnit.MILLISECONDS);
            if (value == null) {
                throw new IllegalStateException("sync callback timeout");
            }
            return value;
        }

        public String awaitOneway(long timeoutMillis) throws Exception {
            String value = onewayValues.poll(timeoutMillis, TimeUnit.MILLISECONDS);
            if (value == null) {
                throw new IllegalStateException("oneway callback timeout");
            }
            return value;
        }
    }

    private static File writeTempFile(String prefix, String content) throws IOException {
        File dir = new File("/data/local/tmp");
        if (!dir.exists() && !dir.mkdirs()) {
            throw new IOException("failed to create temp dir: " + dir);
        }
        File file = File.createTempFile(prefix, ".txt", dir);
        try (FileOutputStream out = new FileOutputStream(file)) {
            out.write(content.getBytes(StandardCharsets.UTF_8));
        }
        return file;
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

    private static void assertServiceSpecificException(RuntimeException error, int wantCode, String wantMessage) throws Exception {
        if (!"android.os.ServiceSpecificException".equals(error.getClass().getName())) {
            throw error;
        }
        Field errorCode = error.getClass().getField("errorCode");
        assertEquals("FailServiceSpecific.code", wantCode, errorCode.getInt(error));
        assertEquals("FailServiceSpecific.message", wantMessage, error.getMessage());
    }
}
