package demo;

interface IFoo {
  const int A = 1 << 0;
  const int B = A | (1 << 1);
}
