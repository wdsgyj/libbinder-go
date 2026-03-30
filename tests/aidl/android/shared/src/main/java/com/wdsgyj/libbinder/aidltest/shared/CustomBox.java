package com.wdsgyj.libbinder.aidltest.shared;

import android.os.Parcel;
import android.os.Parcelable;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;

public final class CustomBox implements Parcelable {
    public static final Creator<CustomBox> CREATOR = new Creator<CustomBox>() {
        @Override
        public CustomBox createFromParcel(Parcel source) {
            return new CustomBox(source);
        }

        @Override
        public CustomBox[] newArray(int size) {
            return new CustomBox[size];
        }
    };

    public int id;
    public String label;
    public List<String> tags;
    public Map<String, String> meta;

    public CustomBox() {
    }

    private CustomBox(Parcel source) {
        id = source.readInt();
        label = source.readString();
        tags = readStringListBody(source);
        meta = readStringMapBody(source);
    }

    @Override
    public int describeContents() {
        return 0;
    }

    @Override
    public void writeToParcel(Parcel dest, int flags) {
        dest.writeInt(id);
        dest.writeString(label);
        writeStringListBody(dest, tags);
        writeStringMapBody(dest, meta);
    }

    public static boolean equalsValue(CustomBox left, CustomBox right) {
        if (left == right) {
            return true;
        }
        if (left == null || right == null) {
            return false;
        }
        return left.id == right.id
                && Objects.equals(left.label, right.label)
                && Objects.equals(left.tags, right.tags)
                && Objects.equals(left.meta, right.meta);
    }

    private static void writeStringListBody(Parcel dest, List<String> value) {
        if (value == null) {
            dest.writeInt(-1);
            return;
        }
        dest.writeInt(value.size());
        for (String item : value) {
            dest.writeString(item);
        }
    }

    private static List<String> readStringListBody(Parcel source) {
        int size = source.readInt();
        if (size < 0) {
            return null;
        }
        ArrayList<String> out = new ArrayList<>(size);
        for (int i = 0; i < size; i++) {
            out.add(source.readString());
        }
        return out;
    }

    private static void writeStringMapBody(Parcel dest, Map<String, String> value) {
        if (value == null) {
            dest.writeInt(-1);
            return;
        }
        dest.writeInt(value.size());
        for (Map.Entry<String, String> entry : value.entrySet()) {
            dest.writeString(entry.getKey());
            dest.writeString(entry.getValue());
        }
    }

    private static Map<String, String> readStringMapBody(Parcel source) {
        int size = source.readInt();
        if (size < 0) {
            return null;
        }
        HashMap<String, String> out = new HashMap<>(size);
        for (int i = 0; i < size; i++) {
            out.put(source.readString(), source.readString());
        }
        return out;
    }
}
