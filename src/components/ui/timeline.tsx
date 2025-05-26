import type React from 'react';
import { cn } from '~/lib/utils';

interface TimelineProps {
	children: React.ReactNode;
	className?: string;
}

export function Timeline({ children, className }: TimelineProps) {
	return (
		<div
			className={cn(
				'relative space-y-8 before:absolute before:inset-0 before:ml-[1.5rem] before:h-full before:w-0.5 before:bg-gradient-to-b before:from-primary/50 before:via-primary/30 before:to-primary/0',
				className
			)}
		>
			{children}
		</div>
	);
}

interface TimelineItemProps {
	year: string;
	children: React.ReactNode;
	className?: string;
}

export function TimelineItem({ year, children, className }: TimelineItemProps) {
	return (
		<div className={cn('relative pl-8 md:pl-12', className)}>
			<div className='flex flex-col md:flex-row md:items-center gap-2 md:gap-4'>
				<div className='absolute left-0 flex h-12 w-12 items-center justify-center rounded-full bg-primary text-primary-foreground text-sm font-bold'>
					{year}
				</div>
				<div className='bg-card rounded-lg p-4 shadow-sm border'>
					{children}
				</div>
			</div>
		</div>
	);
}
