package com.wdsgyj.libbinder.aidltest.javaserver;

import android.app.Service;
import android.content.Intent;
import android.os.IBinder;

public final class PlaceholderService extends Service {
    @Override
    public IBinder onBind(Intent intent) {
        return null;
    }
}
