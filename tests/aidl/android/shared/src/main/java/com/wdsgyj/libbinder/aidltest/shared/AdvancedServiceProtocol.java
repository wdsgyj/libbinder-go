package com.wdsgyj.libbinder.aidltest.shared;

import android.os.Binder;
import android.os.IBinder;
import android.os.IInterface;
import android.os.Parcel;
import android.os.ParcelFileDescriptor;
import android.os.RemoteException;
import java.io.FileDescriptor;
import java.lang.reflect.Method;

public final class AdvancedServiceProtocol {
    public static final String DESCRIPTOR = "com.wdsgyj.libbinder.aidltest.shared.IAdvancedService";
    public static final int TRANSACTION_ECHO_BINDER = IBinder.FIRST_CALL_TRANSACTION + 0;
    public static final int TRANSACTION_INVOKE_CALLBACK = IBinder.FIRST_CALL_TRANSACTION + 1;
    public static final int TRANSACTION_FIRE_ONEWAY = IBinder.FIRST_CALL_TRANSACTION + 2;
    public static final int TRANSACTION_FAIL_SERVICE_SPECIFIC = IBinder.FIRST_CALL_TRANSACTION + 3;
    public static final int TRANSACTION_READ_FROM_FILE_DESCRIPTOR = IBinder.FIRST_CALL_TRANSACTION + 4;
    public static final int TRANSACTION_READ_FROM_PARCEL_FILE_DESCRIPTOR = IBinder.FIRST_CALL_TRANSACTION + 5;

    private static final Method READ_RAW_FILE_DESCRIPTOR = lookupParcelMethod("readRawFileDescriptor");
    private static final Method WRITE_RAW_FILE_DESCRIPTOR = lookupParcelMethod("writeRawFileDescriptor", FileDescriptor.class);

    private AdvancedServiceProtocol() {
    }

    public interface Service extends IInterface {
        IBinder EchoBinder(IBinder input) throws RemoteException;

        String InvokeCallback(IAdvancedCallback callback, String value) throws RemoteException;

        void FireOneway(IAdvancedCallback callback, String value) throws RemoteException;

        void FailServiceSpecific(int code, String message) throws RemoteException;

        String ReadFromFileDescriptor(FileDescriptor fd) throws RemoteException;

        String ReadFromParcelFileDescriptor(ParcelFileDescriptor fd) throws RemoteException;
    }

    public static IBinder newBinder(Service impl) {
        Binder binder = new Binder() {
            @Override
            protected boolean onTransact(int code, Parcel data, Parcel reply, int flags) throws RemoteException {
                if (code >= IBinder.FIRST_CALL_TRANSACTION && code <= IBinder.LAST_CALL_TRANSACTION) {
                    data.enforceInterface(DESCRIPTOR);
                }
                if (code == INTERFACE_TRANSACTION) {
                    reply.writeString(DESCRIPTOR);
                    return true;
                }
                switch (code) {
                    case TRANSACTION_ECHO_BINDER: {
                        IBinder result = impl.EchoBinder(data.readStrongBinder());
                        reply.writeNoException();
                        reply.writeStrongBinder(result);
                        return true;
                    }
                    case TRANSACTION_INVOKE_CALLBACK: {
                        IAdvancedCallback callback = IAdvancedCallback.Stub.asInterface(data.readStrongBinder());
                        String value = data.readString();
                        String result = impl.InvokeCallback(callback, value);
                        reply.writeNoException();
                        reply.writeString(result);
                        return true;
                    }
                    case TRANSACTION_FIRE_ONEWAY: {
                        IAdvancedCallback callback = IAdvancedCallback.Stub.asInterface(data.readStrongBinder());
                        String value = data.readString();
                        impl.FireOneway(callback, value);
                        return true;
                    }
                    case TRANSACTION_FAIL_SERVICE_SPECIFIC: {
                        int serviceCode = data.readInt();
                        String message = data.readString();
                        reply.writeInt(-8);
                        reply.writeString(message);
                        reply.writeInt(0);
                        reply.writeInt(serviceCode);
                        return true;
                    }
                    case TRANSACTION_READ_FROM_FILE_DESCRIPTOR: {
                        String result = impl.ReadFromFileDescriptor(readRawFileDescriptor(data));
                        reply.writeNoException();
                        reply.writeString(result);
                        return true;
                    }
                    case TRANSACTION_READ_FROM_PARCEL_FILE_DESCRIPTOR: {
                        ParcelFileDescriptor value = data.readTypedObject(ParcelFileDescriptor.CREATOR);
                        String result = impl.ReadFromParcelFileDescriptor(value);
                        reply.writeNoException();
                        reply.writeString(result);
                        return true;
                    }
                    default:
                        return super.onTransact(code, data, reply, flags);
                }
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
        public IBinder EchoBinder(IBinder input) throws RemoteException {
            Parcel data = Parcel.obtain();
            Parcel reply = Parcel.obtain();
            try {
                data.writeInterfaceToken(DESCRIPTOR);
                data.writeStrongBinder(input);
                remote.transact(TRANSACTION_ECHO_BINDER, data, reply, 0);
                reply.readException();
                return reply.readStrongBinder();
            } finally {
                reply.recycle();
                data.recycle();
            }
        }

        @Override
        public String InvokeCallback(IAdvancedCallback callback, String value) throws RemoteException {
            Parcel data = Parcel.obtain();
            Parcel reply = Parcel.obtain();
            try {
                data.writeInterfaceToken(DESCRIPTOR);
                data.writeStrongInterface(callback);
                data.writeString(value);
                remote.transact(TRANSACTION_INVOKE_CALLBACK, data, reply, 0);
                reply.readException();
                return reply.readString();
            } finally {
                reply.recycle();
                data.recycle();
            }
        }

        @Override
        public void FireOneway(IAdvancedCallback callback, String value) throws RemoteException {
            Parcel data = Parcel.obtain();
            try {
                data.writeInterfaceToken(DESCRIPTOR);
                data.writeStrongInterface(callback);
                data.writeString(value);
                remote.transact(TRANSACTION_FIRE_ONEWAY, data, null, IBinder.FLAG_ONEWAY);
            } finally {
                data.recycle();
            }
        }

        @Override
        public void FailServiceSpecific(int code, String message) throws RemoteException {
            Parcel data = Parcel.obtain();
            Parcel reply = Parcel.obtain();
            try {
                data.writeInterfaceToken(DESCRIPTOR);
                data.writeInt(code);
                data.writeString(message);
                remote.transact(TRANSACTION_FAIL_SERVICE_SPECIFIC, data, reply, 0);
                reply.readException();
            } finally {
                reply.recycle();
                data.recycle();
            }
        }

        @Override
        public String ReadFromFileDescriptor(FileDescriptor fd) throws RemoteException {
            Parcel data = Parcel.obtain();
            Parcel reply = Parcel.obtain();
            try {
                data.writeInterfaceToken(DESCRIPTOR);
                writeRawFileDescriptor(data, fd);
                remote.transact(TRANSACTION_READ_FROM_FILE_DESCRIPTOR, data, reply, 0);
                reply.readException();
                return reply.readString();
            } finally {
                reply.recycle();
                data.recycle();
            }
        }

        @Override
        public String ReadFromParcelFileDescriptor(ParcelFileDescriptor fd) throws RemoteException {
            Parcel data = Parcel.obtain();
            Parcel reply = Parcel.obtain();
            try {
                data.writeInterfaceToken(DESCRIPTOR);
                data.writeTypedObject(fd, 0);
                remote.transact(TRANSACTION_READ_FROM_PARCEL_FILE_DESCRIPTOR, data, reply, 0);
                reply.readException();
                return reply.readString();
            } finally {
                reply.recycle();
                data.recycle();
            }
        }
    }

    private static Method lookupParcelMethod(String name, Class<?>... parameterTypes) {
        try {
            Method method = Parcel.class.getDeclaredMethod(name, parameterTypes);
            method.setAccessible(true);
            return method;
        } catch (ReflectiveOperationException e) {
            throw new ExceptionInInitializerError(e);
        }
    }

    private static FileDescriptor readRawFileDescriptor(Parcel parcel) throws RemoteException {
        try {
            return (FileDescriptor) READ_RAW_FILE_DESCRIPTOR.invoke(parcel);
        } catch (ReflectiveOperationException e) {
            throw new RemoteException("readRawFileDescriptor reflection failed: " + e);
        }
    }

    private static void writeRawFileDescriptor(Parcel parcel, FileDescriptor fd) throws RemoteException {
        try {
            WRITE_RAW_FILE_DESCRIPTOR.invoke(parcel, fd);
        } catch (ReflectiveOperationException e) {
            throw new RemoteException("writeRawFileDescriptor reflection failed: " + e);
        }
    }
}
