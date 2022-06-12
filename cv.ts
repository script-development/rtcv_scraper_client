export interface CVToScan {
    referenceNumber: string
    link?: string,
    presentation?: string
    personalDetails: CVToScanPersonalDetails
    preferredJobs: Array<string>
    workExperiences: Array<CVToScanWorkExperience>
    educations: Array<CVToScanEducation>
    languages: Array<CVToScanLanguage>
    driversLicenses: Array<string>
}

export function newCvToScan(referenceNumber: string): CVToScan {
    return {
        referenceNumber,
        personalDetails: {},
        preferredJobs: [],
        workExperiences: [],
        educations: [],
        languages: [],
        driversLicenses: [],
    }
}

export interface CVToScanEducation {
    is: 0 | 1 | 2
    name: string
    description: string
    institute: string
    isCompleted?: boolean
    hasDiploma?: boolean
    startDate: string | null // RFC3339
    endDate: string | null // RFC3339
}

export interface CVToScanWorkExperience {
    profession: string
    description: string
    employer: string
    stillEmployed?: boolean
    weeklyHoursWorked?: number
    startDate: string | null // RFC3339
    endDate: string | null // RFC3339
}

export interface CVToScanPersonalDetails {
    city?: string
    country?: string
    dob?: string | null // RFC3339
    email?: string
    firstName?: string
    gender?: string
    houseNumber?: string
    houseNumberSuffix?: string
    initials?: string
    phoneNumber?: string
    streetName?: string
    surName?: string
    surNamePrefix?: string
    zip?: string
}

export interface CVToScanLanguage {
    levelSpoken: LangLevel | null
    levelWritten: LangLevel | null
    name: string
}

export enum LangLevel {
    Unknown = 0,
    Reasonable = 1,
    Good = 2,
    Excellent = 3,
}
