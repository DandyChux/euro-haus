import { useState } from "react"
import { Input } from "~/components/ui/input"
import { Button } from "~/components/ui/button"
import { Search } from "lucide-react"

interface SearchBarProps {
	onSearch: (query: string) => void
}

export function SearchBar({ onSearch }: SearchBarProps) {
	const [searchQuery, setSearchQuery] = useState<string>("")
	const [isExpanded, setIsExpanded] = useState<boolean>(false)

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault()
		onSearch(searchQuery)
	}

	const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
		const newQuery = e.target.value
		setSearchQuery(newQuery)
		onSearch(newQuery) // Real-time filtering as user types
	}

	const toggleSearch = () => {
		setIsExpanded(!isExpanded)
		// Focus the input when expanded
		if (!isExpanded) {
			setTimeout(() => document.getElementById("search-input")?.focus(), 100)
		}
	}

	return (
		<form onSubmit={handleSubmit} className="flex items-center">
			<div className="relative flex items-center">
				{/* Search button first, then input - this helps with the left expansion */}
				<Button
					type="button"
					variant="ghost"
					onClick={toggleSearch}
					className="hover:cursor-pointer"
				>
					<Search className='h-4 w-4' />
				</Button>
				<div className={`absolute right-full overflow-hidden transition-all duration-300 ${isExpanded ? "w-28 sm:w-40 md:w-56 pr-2" : "w-0"}`}>
					<Input
						id="search-input"
						type="text"
						placeholder="Search..."
						value={searchQuery}
						onChange={handleInputChange}
						className='w-full'
					/>
				</div>
			</div>
		</form>
	)
}
