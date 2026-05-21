export async function getSystemInfo(): Promise<{ arch: string; os: string }> {
  return { arch: "x86_64", os: "linux" };
}
