package com.wdsgyj.libbinder.aidltest.javaserver;

import android.os.IBinder;
import com.wdsgyj.libbinder.aidltest.shared.MetadataFixtures;
import com.wdsgyj.libbinder.aidltest.shared.MetadataServiceProtocol;

public final class MetadataServiceImpl implements MetadataServiceProtocol.Service {
    private final String prefix;

    public MetadataServiceImpl(String prefix) {
        this.prefix = prefix;
    }

    @Override
    public String Echo(String value) {
        return prefix + ":" + value;
    }

    @Override
    public int getInterfaceVersion() {
        return MetadataFixtures.VERSION;
    }

    @Override
    public String getInterfaceHash() {
        return MetadataFixtures.HASH;
    }

    @Override
    public IBinder asBinder() {
        return null;
    }
}
