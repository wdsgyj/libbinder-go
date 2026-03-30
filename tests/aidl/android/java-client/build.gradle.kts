plugins {
    id("com.android.application")
}

android {
    namespace = "com.wdsgyj.libbinder.aidltest.javaclient"
    compileSdk = 35
    buildToolsVersion = "35.0.1"

    defaultConfig {
        applicationId = "com.wdsgyj.libbinder.aidltest.javaclient"
        minSdk = 30
        targetSdk = 35
        versionCode = 100
        versionName = "0.1.0"
        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
    }
}

dependencies {
    implementation(project(":shared"))
}
