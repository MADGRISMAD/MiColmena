<script lang="ts">
	import { Badge, type BadgeVariant } from '#lib/components/ui/badge/index.js';
	import type { TicketStatus } from '#lib/api/types.js';
	import { statusLabels } from '#lib/format.js';

	let { status }: { status: TicketStatus } = $props();

	const variants: Record<TicketStatus, BadgeVariant> = {
		open: 'default',
		in_progress: 'secondary',
		waiting: 'outline',
		resolved: 'ghost',
		closed: 'ghost'
	};
</script>

<Badge variant={variants[status]}>{statusLabels[status]}</Badge>
