import Card from "@/components/share/cards/card"
import { useGetUserList } from "@/hooks/api/use-user"

export default function DashboardPage() {
  const { data: userData } = useGetUserList({})

  return (
    <>
      <Card>
        <pre className="overflow-x-auto rounded-xl border bg-muted/30 p-4 text-sm">
          {JSON.stringify(userData, null, 2)}
        </pre>
      </Card>

      <div className="grid auto-rows-min gap-4 md:grid-cols-3">
        <Card>
          <div className="aspect-video rounded-xl" />
        </Card>
        <Card>
          <div className="aspect-video rounded-xl" />
        </Card>
        <Card>
          <div className="aspect-video rounded-xl" />
        </Card>
      </div>
      <div className="grid auto-rows-min gap-4 md:grid-cols-3">
        <Card>
          <div className="aspect-video rounded-xl" />
        </Card>
        <Card>
          <div className="aspect-video rounded-xl" />
        </Card>
        <Card>
          <div className="aspect-video rounded-xl" />
        </Card>
      </div>
      <div className="grid auto-rows-min gap-4 md:grid-cols-3">
        <Card>
          <div className="aspect-video rounded-xl" />
        </Card>
        <Card>
          <div className="aspect-video rounded-xl" />
        </Card>
        <Card>
          <div className="aspect-video rounded-xl" />
        </Card>
      </div>
      <div className="grid auto-rows-min gap-4 md:grid-cols-3">
        <Card>
          <div className="aspect-video rounded-xl" />
        </Card>
        <Card>
          <div className="aspect-video rounded-xl" />
        </Card>
        <Card>
          <div className="aspect-video rounded-xl" />
        </Card>
      </div>
      <div className="grid auto-rows-min gap-4 md:grid-cols-3">
        <div className="aspect-video rounded-xl" />
        <div className="aspect-video rounded-xl" />
        <div className="aspect-video rounded-xl" />
      </div>
      <div className="min-h-[100vh] flex-1 rounded-xl md:min-h-min" />
      <div className="min-h-[100vh] flex-1 rounded-xl md:min-h-min" />
    </>
  )
}
