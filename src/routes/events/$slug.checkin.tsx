import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/events/$slug/checkin')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/events/$slug/checkin"!</div>
}
