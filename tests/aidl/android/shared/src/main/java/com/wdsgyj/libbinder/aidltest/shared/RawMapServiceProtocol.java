package com.wdsgyj.libbinder.aidltest.shared;

import android.os.Binder;
import android.os.IBinder;
import android.os.IInterface;
import android.os.Parcel;
import android.os.RemoteException;
import java.util.HashMap;
import java.util.Map;

public final class RawMapServiceProtocol {
    public static final String DESCRIPTOR = "com.wdsgyj.libbinder.aidltest.shared.IRawMapService";
    public static final int TRANSACTION_NORMALIZE = IBinder.FIRST_CALL_TRANSACTION + 0;

    private RawMapServiceProtocol() {
    }

    public interface Service extends IInterface {
        Map<?, ?> Normalize(Map<?, ?> value) throws RemoteException;
    }

    public static IBinder newBinder(Service impl) {
        Binder binder = new Binder() {
            @Override
            protected boolean onTransact(int code, Parcel data, Parcel reply, int flags) throws RemoteException {
                if (code == INTERFACE_TRANSACTION) {
                    reply.writeString(DESCRIPTOR);
                    return true;
                }
                if (code >= IBinder.FIRST_CALL_TRANSACTION && code <= IBinder.LAST_CALL_TRANSACTION) {
                    data.enforceInterface(DESCRIPTOR);
                }
                if (code == TRANSACTION_NORMALIZE) {
                    Map<?, ?> input = readMapBody(data);
                    Map<?, ?> output = impl.Normalize(input);
                    reply.writeNoException();
                    writeMapBody(reply, output);
                    return true;
                }
                return super.onTransact(code, data, reply, flags);
            }
        };
        binder.attachInterface(impl, DESCRIPTOR);
        return binder;
    }

    public static Service asInterface(IBinder binder) {
        if (binder == null) {
            return null;
        }
        IInterface local = binder.queryLocalInterface(DESCRIPTOR);
        if (local instanceof Service) {
            return (Service) local;
        }
        return new Proxy(binder);
    }

    private static final class Proxy implements Service {
        private final IBinder remote;

        private Proxy(IBinder remote) {
            this.remote = remote;
        }

        @Override
        public IBinder asBinder() {
            return remote;
        }

        @Override
        public Map<?, ?> Normalize(Map<?, ?> value) throws RemoteException {
            Parcel data = Parcel.obtain();
            Parcel reply = Parcel.obtain();
            try {
                data.writeInterfaceToken(DESCRIPTOR);
                writeMapBody(data, value);
                remote.transact(TRANSACTION_NORMALIZE, data, reply, 0);
                reply.readException();
                return readMapBody(reply);
            } finally {
                reply.recycle();
                data.recycle();
            }
        }
    }

    private static void writeMapBody(Parcel parcel, Map<?, ?> value) {
        if (value == null) {
            parcel.writeInt(-1);
            return;
        }
        parcel.writeInt(value.size());
        for (Map.Entry<?, ?> entry : value.entrySet()) {
            parcel.writeValue(entry.getKey());
            parcel.writeValue(entry.getValue());
        }
    }

    private static Map<Object, Object> readMapBody(Parcel parcel) {
        int size = parcel.readInt();
        if (size < 0) {
            return null;
        }
        HashMap<Object, Object> out = new HashMap<>();
        ClassLoader loader = RawMapServiceProtocol.class.getClassLoader();
        for (int i = 0; i < size; i++) {
            Object key = parcel.readValue(loader);
            Object value = parcel.readValue(loader);
            out.put(key, value);
        }
        return out;
    }
}
