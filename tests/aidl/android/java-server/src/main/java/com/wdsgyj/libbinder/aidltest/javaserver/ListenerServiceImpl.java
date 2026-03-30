package com.wdsgyj.libbinder.aidltest.javaserver;

import android.os.IBinder;
import com.wdsgyj.libbinder.aidltest.shared.IListenerCallback;
import com.wdsgyj.libbinder.aidltest.shared.IListenerService;
import com.wdsgyj.libbinder.aidltest.shared.ListenerFixtures;

public final class ListenerServiceImpl extends IListenerService.Stub {
    private final ListenerFixtures.Registry registry = new ListenerFixtures.Registry();

    @Override
    public void RegisterListener(IListenerCallback callback) {
        registry.register(callback);
    }

    @Override
    public void UnregisterListener(IListenerCallback callback) {
        registry.unregister(callback);
    }

    @Override
    public int Emit(String value) throws android.os.RemoteException {
        try {
            return registry.emit(value);
        } catch (Exception e) {
            if (e instanceof android.os.RemoteException) {
                throw (android.os.RemoteException) e;
            }
            throw new RuntimeException(e);
        }
    }

    @Override
    public IBinder EchoBinder(IBinder input) {
        return input;
    }
}
