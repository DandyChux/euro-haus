import { cn } from "~/lib/utils";
import React, { useEffect, useState } from "react";

// Type for testimonial items
export interface TestimonialItem {
	quote: string;
	name: string;
	title: string;
}

// Type for logo items
export interface LogoItem {
	name: string;
	logoUrl: string;
	link?: string;
}

// Union type for both item types
export type InfiniteMovingItem = TestimonialItem | LogoItem;

// Type guard to check if item is a logo
function isLogoItem(item: InfiniteMovingItem): item is LogoItem {
	return 'logoUrl' in item;
}

export const InfiniteMovingCards = ({
	items,
	direction = "left",
	speed = "fast",
	pauseOnHover = true,
	className,
	variant = "testimonial",
}: {
	items: InfiniteMovingItem[];
	direction?: "left" | "right";
	speed?: "fast" | "normal" | "slow";
	pauseOnHover?: boolean;
	className?: string;
	variant?: "testimonial" | "logo";
}) => {
	const containerRef = React.useRef<HTMLDivElement>(null);
	const scrollerRef = React.useRef<HTMLUListElement>(null);

	useEffect(() => {
		addAnimation();
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, []);
	const [start, setStart] = useState(false);
	function addAnimation() {
		if (containerRef.current && scrollerRef.current) {
			const scrollerContent = Array.from(scrollerRef.current.children);

			scrollerContent.forEach((item) => {
				const duplicatedItem = item.cloneNode(true);
				if (scrollerRef.current) {
					scrollerRef.current.appendChild(duplicatedItem);
				}
			});

			getDirection();
			getSpeed();
			setStart(true);
		}
	}
	const getDirection = () => {
		if (containerRef.current) {
			if (direction === "left") {
				containerRef.current.style.setProperty(
					"--animation-direction",
					"forwards",
				);
			} else {
				containerRef.current.style.setProperty(
					"--animation-direction",
					"reverse",
				);
			}
		}
	};
	const getSpeed = () => {
		if (containerRef.current) {
			if (speed === "fast") {
				containerRef.current.style.setProperty("--animation-duration", "20s");
			} else if (speed === "normal") {
				containerRef.current.style.setProperty("--animation-duration", "40s");
			} else {
				containerRef.current.style.setProperty("--animation-duration", "80s");
			}
		}
	};

	const renderItem = (item: InfiniteMovingItem, idx: number) => {
		if (variant === "logo" && isLogoItem(item)) {
			const content = (
				<div className="flex items-center justify-center h-40 w-60 p-4">
					<img
						src={item.logoUrl}
						alt={item.name}
						className="max-h-full max-w-full object-contain filter hover:grayscale transition-all duration-300"
					/>
				</div>
			);

			return (
				<li
					key={item.name + idx}
					className="relative shrink-0 rounded-xl bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800"
				>
					{item.link ? (
						<a href={item.link} target="_blank" rel="noopener noreferrer" className="block">
							{content}
						</a>
					) : (
						content
					)}
				</li>
			);
		}

		// Default testimonial rendering
		const testimonialItem = item as TestimonialItem;
		return (
			<li
				className="relative w-[350px] max-w-full shrink-0 rounded-2xl border border-b-0 border-zinc-200 bg-[linear-gradient(180deg,#fafafa,#f5f5f5)] px-8 py-6 md:w-[450px] dark:border-zinc-700 dark:bg-[linear-gradient(180deg,#27272a,#18181b)]"
				key={testimonialItem.name + idx}
			>
				<blockquote>
					<div
						aria-hidden="true"
						className="user-select-none pointer-events-none absolute -top-0.5 -left-0.5 -z-1 h-[calc(100%_+_4px)] w-[calc(100%_+_4px)]"
					></div>
					<span className="relative z-20 text-sm leading-[1.6] font-normal text-neutral-800 dark:text-gray-100">
						{testimonialItem.quote}
					</span>
					<div className="relative z-20 mt-6 flex flex-row items-center">
						<span className="flex flex-col gap-1">
							<span className="text-sm leading-[1.6] font-normal text-neutral-500 dark:text-gray-400">
								{testimonialItem.name}
							</span>
							<span className="text-sm leading-[1.6] font-normal text-neutral-500 dark:text-gray-400">
								{testimonialItem.title}
							</span>
						</span>
					</div>
				</blockquote>
			</li>
		);
	};

	return (
		<div
			ref={containerRef}
			className={cn(
				"scroller relative z-20 max-w-7xl overflow-hidden [mask-image:linear-gradient(to_right,transparent,white_20%,white_80%,transparent)]",
				className,
			)}
		>
			<ul
				ref={scrollerRef}
				className={cn(
					"flex w-max min-w-full shrink-0 flex-nowrap gap-4 py-4",
					start && "animate-scroll",
					pauseOnHover && "hover:[animation-play-state:paused]",
				)}
			>
				{items.map((item, idx) => renderItem(item, idx))}
			</ul>
		</div>
	);
};
