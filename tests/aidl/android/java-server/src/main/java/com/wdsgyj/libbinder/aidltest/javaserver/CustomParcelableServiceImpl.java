package com.wdsgyj.libbinder.aidltest.javaserver;

import com.wdsgyj.libbinder.aidltest.shared.CustomBox;
import com.wdsgyj.libbinder.aidltest.shared.CustomParcelableFixtures;
import com.wdsgyj.libbinder.aidltest.shared.ICustomParcelableService;

public final class CustomParcelableServiceImpl extends ICustomParcelableService.Stub {
    private final String prefix;

    public CustomParcelableServiceImpl(String prefix) {
        this.prefix = prefix;
    }

    @Override
    public CustomBox Normalize(CustomBox value) {
        return CustomParcelableFixtures.normalize(prefix, value);
    }

    @Override
    public CustomBox NormalizeNullable(CustomBox value) {
        return CustomParcelableFixtures.normalize(prefix, value);
    }
}
