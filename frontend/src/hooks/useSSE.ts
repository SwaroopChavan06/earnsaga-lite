import { useEffect, useRef, useState } from "react";

// Opens an EventSource to `url` and returns the latest parsed JSON value.
// Cleans up the connection when the component unmounts or the url changes.
export function useSSE<T>(url: string): { data: T | null; error: boolean } {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState(false);
  const esRef = useRef<EventSource | null>(null);

  useEffect(() => {
    if (!url) return;

    const es = new EventSource(url);
    esRef.current = es;
    setError(false);

    es.onmessage = (e) => {
      try {
        setData(JSON.parse(e.data) as T);
        setError(false);
      } catch {
        // malformed frame — keep last good value
      }
    };

    es.onerror = () => {
      setError(true);
      es.close();
    };

    return () => {
      es.close();
      esRef.current = null;
    };
  }, [url]);

  return { data, error };
}
