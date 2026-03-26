package demo;

parcelable Holder {
  enum Kind {
    ONE,
    TWO,
  }
  const int Mask = 1 << 3;
  Kind kind = Kind.TWO;
}
