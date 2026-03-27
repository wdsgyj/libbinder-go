package android.content;

import android.content.Intent;
import android.os.Bundle;

oneway interface IIntentReceiver {
    void performReceive(@nullable in Intent intent, int resultCode, @nullable String data,
            @nullable in Bundle extras, boolean ordered, boolean sticky, int sendingUser);
}
