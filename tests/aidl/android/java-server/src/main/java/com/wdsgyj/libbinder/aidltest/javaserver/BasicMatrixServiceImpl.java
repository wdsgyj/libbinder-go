package com.wdsgyj.libbinder.aidltest.javaserver;

import com.wdsgyj.libbinder.aidltest.shared.BasicBundle;
import com.wdsgyj.libbinder.aidltest.shared.BasicEnvelope;
import com.wdsgyj.libbinder.aidltest.shared.BasicMatrixFixtures;
import com.wdsgyj.libbinder.aidltest.shared.BasicMode;
import com.wdsgyj.libbinder.aidltest.shared.BasicStringGroup;
import com.wdsgyj.libbinder.aidltest.shared.BasicUnion;
import com.wdsgyj.libbinder.aidltest.shared.BaselinePayload;
import com.wdsgyj.libbinder.aidltest.shared.IBasicMatrixService;
import java.util.List;
import java.util.Map;

public final class BasicMatrixServiceImpl extends IBasicMatrixService.Stub {
    private final String prefix;

    public BasicMatrixServiceImpl(String prefix) {
        this.prefix = prefix;
    }

    @Override
    public String EchoNullable(String value) {
        return BasicMatrixFixtures.echoNullable(prefix, value);
    }

    @Override
    public int[] ReverseInts(int[] values) {
        return BasicMatrixFixtures.reverseInts(values);
    }

    @Override
    public int[] RotateTriple(int[] triple) {
        return BasicMatrixFixtures.rotateTriple(triple);
    }

    @Override
    public List<String> DecorateTags(List<String> tags) {
        return BasicMatrixFixtures.decorateTags(prefix, tags);
    }

    @Override
    public List<BasicStringGroup> DecorateTagGroups(List<BasicStringGroup> groups) {
        return BasicMatrixFixtures.decorateTagGroups(prefix, groups);
    }

    @Override
    public List<BaselinePayload> DecoratePayloads(List<BaselinePayload> payloads) {
        return BasicMatrixFixtures.decoratePayloads(prefix, payloads);
    }

    @Override
    public Map<String, String> DecorateLabels(Map<String, String> labels) {
        return BasicMatrixFixtures.decorateLabels(prefix, labels);
    }

    @Override
    public Map<String, BaselinePayload> DecoratePayloadMap(Map<String, BaselinePayload> payloadMap) {
        return BasicMatrixFixtures.decoratePayloadMap(prefix, payloadMap);
    }

    @Override
    public Map<String, List<BaselinePayload>> DecoratePayloadBuckets(Map<String, List<BaselinePayload>> payloadBuckets) {
        return BasicMatrixFixtures.decoratePayloadBuckets(prefix, payloadBuckets);
    }

    @Override
    public byte FlipMode(byte mode) {
        return BasicMatrixFixtures.flipMode(mode);
    }

    @Override
    public BasicUnion NormalizeUnion(BasicUnion value) {
        return BasicMatrixFixtures.normalizeUnion(prefix, value);
    }

    @Override
    public BasicBundle NormalizeBundle(BasicBundle value) {
        return BasicMatrixFixtures.normalizeBundle(prefix, value);
    }

    @Override
    public BasicEnvelope NormalizeEnvelope(BasicEnvelope value) {
        return BasicMatrixFixtures.normalizeEnvelope(prefix, value);
    }

    @Override
    public int ExpandBundle(BasicBundle input, BasicBundle doubled, BasicBundle payload) {
        return BasicMatrixFixtures.expandBundle(prefix, input, doubled, payload);
    }
}
