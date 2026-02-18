export function withQuery(path: string, setParams: (params: URLSearchParams) => void): string {
  const params = new URLSearchParams();
  setParams(params);

  const query = params.toString();
  return query ? `${path}?${query}` : path;
}
