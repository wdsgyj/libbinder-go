package com.wdsgyj.libbinder.aidltest.shared;

import android.os.IBinder;
import android.os.Parcel;

public final class MetadataFixtures {
    public static final int VERSION = 7;
    public static final String HASH = "metadata-hash-v7";

    private MetadataFixtures() {
    }

    public static void verifyService(IBinder binder, MetadataServiceProtocol.Service service, String expectedPrefix) throws Exception {
        verifyUnknownTransaction(binder);
        assertEquals("Echo", expectedPrefix + ":hello", service.Echo("hello"));
        assertEquals("InterfaceVersion", VERSION, service.getInterfaceVersion());
        assertEquals("InterfaceHash", HASH, service.getInterfaceHash());
    }

    public static void verifyUnknownTransaction(IBinder binder) throws Exception {
        Parcel data = Parcel.obtain();
        Parcel reply = Parcel.obtain();
        try {
            data.writeInterfaceToken(MetadataServiceProtocol.DESCRIPTOR);
            boolean handled = binder.transact(IBinder.FIRST_CALL_TRANSACTION + 99, data, reply, 0);
            if (handled) {
                throw new AssertionError("unknown transaction unexpectedly handled");
            }
        } finally {
            reply.recycle();
            data.recycle();
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
}
