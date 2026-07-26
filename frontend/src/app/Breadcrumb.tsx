import { getNavigationItem, type ApplicationPage } from "./navigation";

export function Breadcrumb({ activePage }: { activePage: ApplicationPage }) {
  const item = getNavigationItem(activePage);

  return (
    <nav aria-label="Breadcrumb">
      <ol className="flex items-center gap-2 text-sm font-bold">
        <li className="text-muted">{item.groupLabel}</li>
        <li aria-hidden="true" className="text-disabled">
          /
        </li>
        <li>
          <h1 className="text-text-primary" aria-current="page">
            {item.label}
          </h1>
        </li>
      </ol>
    </nav>
  );
}
