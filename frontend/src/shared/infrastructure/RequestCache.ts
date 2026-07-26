export class RequestCache<T> {
  private request?: Promise<T>;
  private value?: T;
  private hasValue = false;
  private version = 0;

  run(load: () => Promise<T>, signal?: AbortSignal): Promise<T> {
    if (this.hasValue) {
      return waitForConsumer(Promise.resolve(this.value as T), signal);
    }
    if (!this.request) {
      const requestVersion = this.version;
      const request = load()
        .then((value) => {
          if (this.version === requestVersion) {
            this.value = value;
            this.hasValue = true;
          }
          return value;
        })
        .finally(() => {
          if (this.request === request) {
            this.request = undefined;
          }
        });
      this.request = request;
    }
    return waitForConsumer(this.request, signal);
  }

  invalidate(): void {
    this.version += 1;
    this.request = undefined;
    this.value = undefined;
    this.hasValue = false;
  }
}

function waitForConsumer<T>(
  request: Promise<T>,
  signal?: AbortSignal,
): Promise<T> {
  if (!signal) return request;
  if (signal.aborted) return Promise.reject(abortError());

  return new Promise<T>((resolve, reject) => {
    const abort = () => reject(abortError());
    signal.addEventListener("abort", abort, { once: true });
    request.then(resolve, reject).finally(() => {
      signal.removeEventListener("abort", abort);
    });
  });
}

function abortError(): DOMException {
  return new DOMException("The operation was aborted", "AbortError");
}
