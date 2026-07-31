import { useCallback, useEffect, useRef, useState } from "react";
import { Alert } from "../../../shared/presentation/Alert";
import { Button } from "../../../shared/presentation/Button";
import { ListSkeleton } from "../../../shared/presentation/ListSkeleton";
import type { WBSGateway } from "../application/wbsGateway";
import type { WBSNode } from "../domain/wbs";
import { summarizeWBS } from "../domain/wbsSummary";
import { WBSSummary } from "./WBSSummary";

export function ProjectWBSSummary({
  projectId,
  gateway,
}: {
  projectId: string;
  gateway: Pick<WBSGateway, "tree" | "subscribeToConfirmedChanges">;
}) {
  const [roots, setRoots] = useState<WBSNode[]>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [reload, setReload] = useState(0);
  const latestRequest = useRef(0);

  const retry = useCallback(() => setReload((value) => value + 1), []);

  useEffect(
    () => gateway.subscribeToConfirmedChanges?.(retry),
    [gateway, retry],
  );

  useEffect(() => {
    const controller = new AbortController();
    const requestID = latestRequest.current + 1;
    latestRequest.current = requestID;
    setLoading(true);
    setError("");

    void gateway
      .tree(projectId, controller.signal)
      .then((value) => {
        if (latestRequest.current === requestID && !controller.signal.aborted) {
          setRoots(value);
          setLoading(false);
        }
      })
      .catch((reason: unknown) => {
        if (
          latestRequest.current === requestID &&
          !controller.signal.aborted &&
          !(reason instanceof DOMException && reason.name === "AbortError")
        ) {
          setError("Project summary could not be loaded. Try again.");
          setLoading(false);
        }
      });

    return () => controller.abort();
  }, [gateway, projectId, reload]);

  if (error) {
    return (
      <div className="rounded-panel border border-border-subtle p-4">
        <Alert tone="danger">{error}</Alert>
        <Button type="button" className="mt-4" onClick={retry}>
          Retry
        </Button>
      </div>
    );
  }

  if (loading || !roots) {
    return <ListSkeleton label="Loading project summary" rows={3} />;
  }

  return (
    <WBSSummary
      summary={summarizeWBS(roots)}
      subject="project"
      idPrefix={`project-${projectId}-summary`}
    />
  );
}
