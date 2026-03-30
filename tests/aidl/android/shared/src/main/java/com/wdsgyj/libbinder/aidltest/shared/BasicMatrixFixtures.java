package com.wdsgyj.libbinder.aidltest.shared;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;

public final class BasicMatrixFixtures {
    private BasicMatrixFixtures() {
    }

    public static BasicBundle inputBundle() {
        BasicBundle value = new BasicBundle();
        value.ints = new int[] {1, 2, 3, 4};
        value.triple = new int[] {7, 8, 9};
        value.note = "seed";
        value.tags = new ArrayList<>(Arrays.asList("red", "blue"));
        value.payloads = new ArrayList<>();
        value.payloads.add(payload(10, "ten"));
        value.payloads.add(payload(20, null));
        value.labels = new HashMap<>();
        value.labels.put("left", "west");
        value.labels.put("right", "east");
        value.payloadMap = new HashMap<>();
        value.payloadMap.put("first", payload(100, "alpha"));
        value.payloadMap.put("second", payload(200, null));
        value.mode = BasicMode.ALPHA;
        value.value = BasicUnion.text("seed");
        return value;
    }

    public static BasicBundle secondBundle() {
        BasicBundle value = new BasicBundle();
        value.ints = new int[] {9, 8};
        value.triple = new int[] {4, 5, 6};
        value.note = null;
        value.tags = new ArrayList<>(Arrays.asList("amber"));
        value.payloads = new ArrayList<>();
        value.payloads.add(payload(3, "bee"));
        value.labels = new HashMap<>();
        value.labels.put("up", "north");
        value.payloadMap = new HashMap<>();
        value.payloadMap.put("solo", payload(33, "solo"));
        value.mode = BasicMode.BETA;
        value.value = BasicUnion.number(41);
        return value;
    }

    public static List<BasicStringGroup> tagGroups() {
        ArrayList<BasicStringGroup> out = new ArrayList<>();
        out.add(group("red", "blue"));
        out.add(group("amber"));
        return out;
    }

    public static Map<String, List<BaselinePayload>> payloadBuckets() {
        HashMap<String, List<BaselinePayload>> out = new HashMap<>();
        out.put("left", new ArrayList<>(Arrays.asList(payload(1, "alpha"), payload(2, null))));
        out.put("right", new ArrayList<>(Arrays.asList(payload(3, "beta"))));
        return out;
    }

    public static BasicEnvelope inputEnvelope() {
        BasicEnvelope value = new BasicEnvelope();
        value.note = null;
        value.primary = payload(5, "prime");
        value.history = new ArrayList<>(Arrays.asList(payload(7, "history"), payload(8, null)));
        value.bundle = inputBundle();
        return value;
    }

    public static String echoNullable(String prefix, String value) {
        return value == null ? null : prefix + ":" + value;
    }

    public static int[] reverseInts(int[] values) {
        if (values == null) {
            return null;
        }
        int[] out = Arrays.copyOf(values, values.length);
        for (int i = 0, j = out.length - 1; i < j; i++, j--) {
            int tmp = out[i];
            out[i] = out[j];
            out[j] = tmp;
        }
        return out;
    }

    public static int[] rotateTriple(int[] value) {
        return new int[] {value[1], value[2], value[0]};
    }

    public static List<String> decorateTags(String prefix, List<String> tags) {
        if (tags == null) {
            return null;
        }
        ArrayList<String> out = new ArrayList<>(tags.size());
        for (String tag : tags) {
            out.add(prefix + ":" + tag);
        }
        return out;
    }

    public static List<BasicStringGroup> decorateTagGroups(String prefix, List<BasicStringGroup> groups) {
        if (groups == null) {
            return null;
        }
        ArrayList<BasicStringGroup> out = new ArrayList<>(groups.size());
        for (BasicStringGroup group : groups) {
            BasicStringGroup value = new BasicStringGroup();
            value.tags = decorateTags(prefix, group.tags);
            out.add(value);
        }
        return out;
    }

    public static List<BaselinePayload> decoratePayloads(String prefix, List<BaselinePayload> payloads) {
        if (payloads == null) {
            return null;
        }
        ArrayList<BaselinePayload> out = new ArrayList<>(payloads.size());
        for (int i = 0; i < payloads.size(); i++) {
            out.add(decoratePayload(prefix, payloads.get(i), i + 1));
        }
        return out;
    }

    public static Map<String, String> decorateLabels(String prefix, Map<String, String> labels) {
        if (labels == null) {
            return null;
        }
        HashMap<String, String> out = new HashMap<>();
        for (Map.Entry<String, String> entry : labels.entrySet()) {
            out.put(entry.getKey(), prefix + ":" + entry.getValue());
        }
        return out;
    }

    public static Map<String, BaselinePayload> decoratePayloadMap(String prefix, Map<String, BaselinePayload> payloadMap) {
        if (payloadMap == null) {
            return null;
        }
        HashMap<String, BaselinePayload> out = new HashMap<>();
        for (Map.Entry<String, BaselinePayload> entry : payloadMap.entrySet()) {
            out.put(entry.getKey(), decoratePayload(prefix, entry.getValue(), entry.getKey().length()));
        }
        return out;
    }

    public static Map<String, List<BaselinePayload>> decoratePayloadBuckets(String prefix, Map<String, List<BaselinePayload>> payloadBuckets) {
        if (payloadBuckets == null) {
            return null;
        }
        HashMap<String, List<BaselinePayload>> out = new HashMap<>();
        for (Map.Entry<String, List<BaselinePayload>> entry : payloadBuckets.entrySet()) {
            out.put(entry.getKey(), decoratePayloads(prefix + ":" + entry.getKey(), entry.getValue()));
        }
        return out;
    }

    public static byte flipMode(byte mode) {
        if (mode == BasicMode.ALPHA) {
            return BasicMode.BETA;
        }
        if (mode == BasicMode.BETA) {
            return BasicMode.ALPHA;
        }
        return BasicMode.ALPHA;
    }

    public static BasicUnion normalizeUnion(String prefix, BasicUnion value) {
        if (value == null) {
            return BasicUnion.text(prefix + ":default");
        }
        switch (value.getTag()) {
            case BasicUnion.number:
                return BasicUnion.number(value.getNumber() + 1);
            case BasicUnion.text:
                return BasicUnion.text(prefixOrDefault(prefix, value.getText(), "default"));
            case BasicUnion.payload:
                return BasicUnion.payload(decoratePayload(prefix, value.getPayload(), 50));
            default:
                return BasicUnion.text(prefix + ":default");
        }
    }

    public static BasicBundle normalizeBundle(String prefix, BasicBundle value) {
        if (value == null) {
            return null;
        }
        BasicBundle out = new BasicBundle();
        out.ints = reverseInts(value.ints);
        out.triple = rotateTriple(value.triple);
        out.note = prefixOrDefault(prefix, value.note, "default");
        out.tags = decorateTags(prefix, value.tags);
        out.payloads = decoratePayloads(prefix, value.payloads);
        out.labels = decorateLabels(prefix, value.labels);
        out.payloadMap = decoratePayloadMap(prefix, value.payloadMap);
        out.mode = flipMode(value.mode);
        out.value = normalizeUnion(prefix, value.value);
        return out;
    }

    public static BasicEnvelope normalizeEnvelope(String prefix, BasicEnvelope value) {
        if (value == null) {
            return null;
        }
        BasicEnvelope out = new BasicEnvelope();
        out.title = prefixOrDefault(prefix, value.title, "untitled");
        out.note = prefixOrDefault(prefix, value.note, "default");
        out.primary = value.primary == null ? null : decoratePayload(prefix, value.primary, 11);
        out.history = decoratePayloads(prefix, value.history);
        out.bundle = normalizeBundle(prefix, value.bundle);
        return out;
    }

    public static int expandBundle(String prefix, BasicBundle input, BasicBundle doubled, BasicBundle payload) {
        int ret = input.ints.length + payload.tags.size();
        copyBundle(normalizeBundle(prefix, input), doubled);
        doubled.ints = append(doubled.ints, ret);
        copyBundle(normalizeBundle(prefix, payload), payload);
        payload.triple = new int[] {payload.triple[0] + ret, payload.triple[1], payload.triple[2]};
        return ret;
    }

    public static void verifyService(IBasicMatrixService service, String prefix) throws Exception {
        BasicBundle input = inputBundle();

        assertEquals("EchoNullable", echoNullable(prefix, "hello"), service.EchoNullable("hello"));
        assertTrue("ReverseInts", Arrays.equals(reverseInts(input.ints), service.ReverseInts(input.ints)));
        assertTrue("RotateTriple", Arrays.equals(rotateTriple(input.triple), service.RotateTriple(input.triple)));
        assertTrue("DecorateTags", equalStringList(decorateTags(prefix, input.tags), service.DecorateTags(input.tags)));
        assertTrue("DecorateTagGroups", equalStringGroupList(decorateTagGroups(prefix, tagGroups()), service.DecorateTagGroups(tagGroups())));
        assertTrue("DecoratePayloads", equalPayloadList(decoratePayloads(prefix, input.payloads), service.DecoratePayloads(input.payloads)));
        assertTrue("DecorateLabels", equalStringMap(decorateLabels(prefix, input.labels), service.DecorateLabels(input.labels)));
        assertTrue("DecoratePayloadMap", equalPayloadMap(decoratePayloadMap(prefix, input.payloadMap), service.DecoratePayloadMap(input.payloadMap)));
        assertTrue("DecoratePayloadBuckets", equalPayloadBucketMap(decoratePayloadBuckets(prefix, payloadBuckets()), service.DecoratePayloadBuckets(payloadBuckets())));
        assertEquals("FlipMode", flipMode(input.mode), service.FlipMode(input.mode));
        assertTrue("NormalizeUnion", equalUnion(normalizeUnion(prefix, input.value), service.NormalizeUnion(input.value)));
        assertTrue("NormalizeBundle", equalBundle(normalizeBundle(prefix, input), service.NormalizeBundle(input)));
        assertTrue("NormalizeEnvelope", equalEnvelope(normalizeEnvelope(prefix, inputEnvelope()), service.NormalizeEnvelope(inputEnvelope())));

        BasicBundle second = secondBundle();
        BasicBundle doubled = new BasicBundle();
        BasicBundle payload = secondBundle();
        int ret = service.ExpandBundle(input, doubled, payload);

        BasicBundle wantDoubled = new BasicBundle();
        BasicBundle wantPayload = secondBundle();
        int wantRet = expandBundle(prefix, inputBundle(), wantDoubled, wantPayload);

        assertEquals("ExpandBundle.ret", wantRet, ret);
        assertTrue("ExpandBundle.doubled", equalBundle(wantDoubled, doubled));
        assertTrue("ExpandBundle.payload", equalBundle(wantPayload, payload));
    }

    public static void verifyLargePayloadService(IBasicMatrixService service, String prefix) throws Exception {
        BasicBundle input = largeInputBundle();

        assertTrue("ReverseInts.large", Arrays.equals(reverseInts(input.ints), service.ReverseInts(input.ints)));
        assertTrue("NormalizeEnvelope.large", equalEnvelope(normalizeEnvelope(prefix, largeEnvelope()), service.NormalizeEnvelope(largeEnvelope())));
        assertTrue("DecoratePayloadBuckets.large", equalPayloadBucketMap(decoratePayloadBuckets(prefix, largePayloadBuckets()), service.DecoratePayloadBuckets(largePayloadBuckets())));
    }

    public static boolean equalBundle(BasicBundle left, BasicBundle right) {
        if (left == right) {
            return true;
        }
        if (left == null || right == null) {
            return false;
        }
        return Arrays.equals(left.ints, right.ints)
                && Arrays.equals(left.triple, right.triple)
                && Objects.equals(left.note, right.note)
                && equalStringList(left.tags, right.tags)
                && equalPayloadList(left.payloads, right.payloads)
                && equalStringMap(left.labels, right.labels)
                && equalPayloadMap(left.payloadMap, right.payloadMap)
                && left.mode == right.mode
                && equalUnion(left.value, right.value);
    }

    public static boolean equalUnion(BasicUnion left, BasicUnion right) {
        if (left == right) {
            return true;
        }
        if (left == null || right == null) {
            return false;
        }
        if (left.getTag() != right.getTag()) {
            return false;
        }
        switch (left.getTag()) {
            case BasicUnion.number:
                return left.getNumber() == right.getNumber();
            case BasicUnion.text:
                return Objects.equals(left.getText(), right.getText());
            case BasicUnion.payload:
                return equalPayload(left.getPayload(), right.getPayload());
            default:
                return false;
        }
    }

    public static boolean equalPayload(BaselinePayload left, BaselinePayload right) {
        if (left == right) {
            return true;
        }
        if (left == null || right == null) {
            return false;
        }
        return left.code == right.code && Objects.equals(left.note, right.note);
    }

    public static boolean equalEnvelope(BasicEnvelope left, BasicEnvelope right) {
        if (left == right) {
            return true;
        }
        if (left == null || right == null) {
            return false;
        }
        return Objects.equals(left.title, right.title)
                && Objects.equals(left.note, right.note)
                && equalPayload(left.primary, right.primary)
                && equalPayloadList(left.history, right.history)
                && equalBundle(left.bundle, right.bundle);
    }

    public static boolean equalPayloadList(List<BaselinePayload> left, List<BaselinePayload> right) {
        if (left == right) {
            return true;
        }
        if (left == null || right == null || left.size() != right.size()) {
            return false;
        }
        for (int i = 0; i < left.size(); i++) {
            if (!equalPayload(left.get(i), right.get(i))) {
                return false;
            }
        }
        return true;
    }

    public static boolean equalStringGroupList(List<BasicStringGroup> left, List<BasicStringGroup> right) {
        if (left == right) {
            return true;
        }
        if (left == null || right == null || left.size() != right.size()) {
            return false;
        }
        for (int i = 0; i < left.size(); i++) {
            if (!equalStringList(left.get(i).tags, right.get(i).tags)) {
                return false;
            }
        }
        return true;
    }

    public static boolean equalPayloadBucketMap(Map<String, List<BaselinePayload>> left, Map<String, List<BaselinePayload>> right) {
        if (left == right) {
            return true;
        }
        if (left == null || right == null || left.size() != right.size()) {
            return false;
        }
        for (Map.Entry<String, List<BaselinePayload>> entry : left.entrySet()) {
            if (!equalPayloadList(entry.getValue(), right.get(entry.getKey()))) {
                return false;
            }
        }
        return true;
    }

    public static boolean equalStringList(List<String> left, List<String> right) {
        return Objects.equals(left, right);
    }

    public static boolean equalStringMap(Map<String, String> left, Map<String, String> right) {
        return Objects.equals(left, right);
    }

    public static boolean equalPayloadMap(Map<String, BaselinePayload> left, Map<String, BaselinePayload> right) {
        if (left == right) {
            return true;
        }
        if (left == null || right == null || left.size() != right.size()) {
            return false;
        }
        for (Map.Entry<String, BaselinePayload> entry : left.entrySet()) {
            if (!equalPayload(entry.getValue(), right.get(entry.getKey()))) {
                return false;
            }
        }
        return true;
    }

    private static BaselinePayload payload(int code, String note) {
        BaselinePayload payload = new BaselinePayload();
        payload.code = code;
        payload.note = note;
        return payload;
    }

    private static BasicStringGroup group(String... values) {
        BasicStringGroup out = new BasicStringGroup();
        out.tags = new ArrayList<>(Arrays.asList(values));
        return out;
    }

    private static BaselinePayload decoratePayload(String prefix, BaselinePayload payload, int codeDelta) {
        BaselinePayload out = new BaselinePayload();
        out.code = payload.code + codeDelta;
        out.note = prefixOrDefault(prefix, payload.note, "default");
        return out;
    }

    private static String prefixOrDefault(String prefix, String value, String fallback) {
        return prefix + ":" + (value == null ? fallback : value);
    }

    private static void copyBundle(BasicBundle src, BasicBundle dst) {
        dst.ints = src.ints == null ? null : Arrays.copyOf(src.ints, src.ints.length);
        dst.triple = src.triple == null ? null : Arrays.copyOf(src.triple, src.triple.length);
        dst.note = src.note;
        dst.tags = src.tags == null ? null : new ArrayList<>(src.tags);
        dst.payloads = src.payloads == null ? null : new ArrayList<>(src.payloads.size());
        if (src.payloads != null) {
            for (BaselinePayload payload : src.payloads) {
                dst.payloads.add(payload(payload.code, payload.note));
            }
        }
        dst.labels = src.labels == null ? null : new HashMap<>(src.labels);
        dst.payloadMap = src.payloadMap == null ? null : new HashMap<>();
        if (src.payloadMap != null) {
            for (Map.Entry<String, BaselinePayload> entry : src.payloadMap.entrySet()) {
                dst.payloadMap.put(entry.getKey(), payload(entry.getValue().code, entry.getValue().note));
            }
        }
        dst.mode = src.mode;
        if (src.value == null) {
            dst.value = null;
        } else {
            switch (src.value.getTag()) {
                case BasicUnion.number:
                    dst.value = BasicUnion.number(src.value.getNumber());
                    break;
                case BasicUnion.text:
                    dst.value = BasicUnion.text(src.value.getText());
                    break;
                case BasicUnion.payload:
                    BaselinePayload payload = src.value.getPayload();
                    dst.value = BasicUnion.payload(payload(payload.code, payload.note));
                    break;
                default:
                    throw new IllegalStateException("unknown tag " + src.value.getTag());
            }
        }
    }

    private static int[] append(int[] values, int value) {
        int[] out = Arrays.copyOf(values, values.length + 1);
        out[out.length - 1] = value;
        return out;
    }

    private static BasicBundle largeInputBundle() {
        BasicBundle value = new BasicBundle();
        value.ints = new int[4096];
        for (int i = 0; i < value.ints.length; i++) {
            value.ints[i] = i + 1;
        }
        value.triple = new int[] {101, 202, 303};
        value.note = "bulk";
        value.tags = new ArrayList<>();
        for (int i = 0; i < 256; i++) {
            value.tags.add(String.format("tag-%03d", i));
        }
        value.payloads = new ArrayList<>();
        for (int i = 0; i < 256; i++) {
            value.payloads.add(payload(10 + i, String.format("note-%03d", i)));
        }
        value.labels = new HashMap<>();
        value.payloadMap = new HashMap<>();
        for (int i = 0; i < 128; i++) {
            String key = String.format("key-%03d", i);
            value.labels.put(key, String.format("label-%03d", i));
            value.payloadMap.put(key, payload(200 + i, String.format("payload-%03d", i)));
        }
        value.mode = BasicMode.ALPHA;
        value.value = BasicUnion.payload(payload(999, "union"));
        return value;
    }

    private static Map<String, List<BaselinePayload>> largePayloadBuckets() {
        HashMap<String, List<BaselinePayload>> out = new HashMap<>();
        for (int i = 0; i < 32; i++) {
            String key = String.format("bucket-%02d", i);
            ArrayList<BaselinePayload> values = new ArrayList<>();
            for (int j = 0; j < 32; j++) {
                values.add(payload(i * 100 + j, String.format("%s-item-%02d", key, j)));
            }
            out.put(key, values);
        }
        return out;
    }

    private static BasicEnvelope largeEnvelope() {
        BasicEnvelope value = new BasicEnvelope();
        value.title = "bulk";
        value.note = "bulk-note";
        value.primary = payload(77, "primary");
        value.history = new ArrayList<>();
        for (int i = 0; i < 128; i++) {
            value.history.add(payload(300 + i, String.format("history-%03d", i)));
        }
        value.bundle = largeInputBundle();
        return value;
    }

    private static void assertEquals(String name, Object want, Object got) {
        if (!Objects.equals(want, got)) {
            throw new IllegalStateException(name + " mismatch: want=" + want + " got=" + got);
        }
    }

    private static void assertTrue(String name, boolean ok) {
        if (!ok) {
            throw new IllegalStateException(name + " mismatch");
        }
    }
}
