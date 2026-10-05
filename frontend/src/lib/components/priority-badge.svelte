<script lang="ts">
	import { Badge, type BadgeVariant } from '#lib/components/ui/badge/index.js';
	import type { TicketPriority } from '#lib/api/types.js';
	import { priorityLabels } from '#lib/format.js';

	let { priority }: { priority: TicketPriority } = $props();

	const variants: Record<TicketPriority, BadgeVariant> = {
		low: 'ghost',
		medium: 'outline',
		high: 'secondary',
		urgent: 'destructive'
	};
</script>

<Badge variant={variants[priority]}>{priorityLabels[priority]}</Badge>
