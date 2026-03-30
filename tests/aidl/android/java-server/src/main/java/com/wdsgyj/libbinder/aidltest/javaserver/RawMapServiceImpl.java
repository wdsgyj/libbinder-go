package com.wdsgyj.libbinder.aidltest.javaserver;

import android.os.IBinder;
import com.wdsgyj.libbinder.aidltest.shared.RawMapFixtures;
import com.wdsgyj.libbinder.aidltest.shared.RawMapServiceProtocol;
import java.util.Map;

public final class RawMapServiceImpl implements RawMapServiceProtocol.Service {
    private final String prefix;

    private RawMapServiceImpl(String prefix) {
        this.prefix = prefix;
    }

    public static IBinder newBinder(String prefix) {
        return RawMapServiceProtocol.newBinder(new RawMapServiceImpl(prefix));
    }

    @Override
    public IBinder asBinder() {
        return null;
    }

    @Override
    public Map<?, ?> Normalize(Map<?, ?> value) {
        return RawMapFixtures.normalize(prefix, value);
    }
}
