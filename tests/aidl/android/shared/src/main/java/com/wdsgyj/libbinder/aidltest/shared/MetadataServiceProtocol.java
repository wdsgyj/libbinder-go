package com.wdsgyj.libbinder.aidltest.shared;

import android.os.Binder;
import android.os.IBinder;
import android.os.IInterface;
import android.os.Parcel;
import android.os.RemoteException;

public final class MetadataServiceProtocol {
    public static final String DESCRIPTOR = "com.wdsgyj.libbinder.aidltest.shared.IMetadataService";
    public static final int TRANSACTION_ECHO = IBinder.FIRST_CALL_TRANSACTION + 0;
    public static final int TRANSACTION_GET_INTERFACE_HASH = 0x00fffffd;
    public static final int TRANSACTION_GET_INTERFACE_VERSION = 0x00fffffe;

    private MetadataServiceProtocol() {
    }

    public interface Service extends IInterface {
        String Echo(String value) throws RemoteException;

        int getInterfaceVersion() throws RemoteException;

        String getInterfaceHash() throws RemoteException;
    }

    public static IBinder newBinder(Service impl) {
        Binder binder = new Binder() {
            @Override
            protected boolean onTransact(int code, Parcel data, Parcel reply, int flags) throws RemoteException {
                if (code == INTERFACE_TRANSACTION) {
                    reply.writeString(DESCRIPTOR);
                    return true;
                }
                if (code == TRANSACTION_GET_INTERFACE_VERSION) {
                    reply.writeInt(impl.getInterfaceVersion());
                    return true;
                }
                if (code == TRANSACTION_GET_INTERFACE_HASH) {
                    reply.writeString(impl.getInterfaceHash());
                    return true;
                }
                if (code >= IBinder.FIRST_CALL_TRANSACTION && code <= IBinder.LAST_CALL_TRANSACTION) {
                    data.enforceInterface(DESCRIPTOR);
                }
                if (code == TRANSACTION_ECHO) {
                    String result = impl.Echo(data.readString());
                    reply.writeNoException();
                    reply.writeString(result);
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
        public String Echo(String value) throws RemoteException {
            Parcel data = Parcel.obtain();
            Parcel reply = Parcel.obtain();
            try {
                data.writeInterfaceToken(DESCRIPTOR);
                data.writeString(value);
                remote.transact(TRANSACTION_ECHO, data, reply, 0);
                reply.readException();
                return reply.readString();
            } finally {
                reply.recycle();
                data.recycle();
            }
        }

        @Override
        public int getInterfaceVersion() throws RemoteException {
            Parcel data = Parcel.obtain();
            Parcel reply = Parcel.obtain();
            try {
                remote.transact(TRANSACTION_GET_INTERFACE_VERSION, data, reply, 0);
                return reply.readInt();
            } finally {
                reply.recycle();
                data.recycle();
            }
        }

        @Override
        public String getInterfaceHash() throws RemoteException {
            Parcel data = Parcel.obtain();
            Parcel reply = Parcel.obtain();
            try {
                remote.transact(TRANSACTION_GET_INTERFACE_HASH, data, reply, 0);
                return reply.readString();
            } finally {
                reply.recycle();
                data.recycle();
            }
        }
    }
}
