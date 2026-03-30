package com.wdsgyj.libbinder.aidltest.javaserver;

import android.os.IBinder;
import android.os.ParcelFileDescriptor;
import android.os.RemoteException;
import com.wdsgyj.libbinder.aidltest.shared.AdvancedFixtures;
import com.wdsgyj.libbinder.aidltest.shared.AdvancedServiceProtocol;
import com.wdsgyj.libbinder.aidltest.shared.IAdvancedCallback;
import java.io.FileDescriptor;
import java.io.IOException;

public final class AdvancedServiceImpl implements AdvancedServiceProtocol.Service {
    private final String prefix;

    public AdvancedServiceImpl(String prefix) {
        this.prefix = prefix;
    }

    @Override
    public IBinder asBinder() {
        return null;
    }

    @Override
    public IBinder EchoBinder(IBinder input) {
        return AdvancedFixtures.echoBinder(input);
    }

    @Override
    public String InvokeCallback(IAdvancedCallback callback, String value) throws RemoteException {
        return AdvancedFixtures.invokeCallback(prefix, callback, value);
    }

    @Override
    public void FireOneway(IAdvancedCallback callback, String value) throws RemoteException {
        AdvancedFixtures.fireOneway(prefix, callback, value);
    }

    @Override
    public void FailServiceSpecific(int code, String message) throws RemoteException {
        throw new RemoteException("unexpected direct FailServiceSpecific invocation");
    }

    @Override
    public String ReadFromFileDescriptor(FileDescriptor fd) {
        try {
            return AdvancedFixtures.readAll(fd);
        } catch (IOException e) {
            throw new RuntimeException(e);
        }
    }

    @Override
    public String ReadFromParcelFileDescriptor(ParcelFileDescriptor fd) {
        try {
            return AdvancedFixtures.readAll(fd);
        } catch (IOException e) {
            throw new RuntimeException(e);
        }
    }
}
