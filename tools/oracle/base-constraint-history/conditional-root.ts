// Research reduction of captured router-core link.d.ts:139,145.
// Route lookups are reduced leaves; Path is explicit to exclude relative-path
// computation. This is not an application reproduction or a diagnostic baseline.
interface AnyRoute {
  types: { allParams: string; fullSearchSchemaInput: number };
}
interface AnyRouter {
  routeTree: AnyRoute;
}
type RouteByPath<TRouteTree extends AnyRoute, TPath> =
  Extract<TRouteTree, AnyRoute>;
type RouteByToPath<TRouter extends AnyRouter, TPath> =
  Extract<TRouter["routeTree"], AnyRoute>;
type ResolveRoute<TRouter extends AnyRouter, TFrom, TTo, TPath> =
  TPath extends string
    ? TFrom extends TPath
      ? RouteByPath<TRouter["routeTree"], TPath>
      : RouteByToPath<TRouter, TPath>
    : never;
// The test reads the last two declarations without checking file semantics.
type Indexed<TRouter extends AnyRouter, TFrom, TTo, TPath> =
  ResolveRoute<TRouter, TFrom, TTo, TPath>["types"];
type ExpectedPayload = { allParams: string; fullSearchSchemaInput: number };
