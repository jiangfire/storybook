export interface ProjectMemberItem {
  id: number;
  user_id: number;
  role_in_project: string;
  joined_at: string;
  is_owner: boolean;
  user?: {
    id: number;
    email: string;
    avatar_url?: string;
  };
}

export type ConfirmActionState =
  | {
      kind: 'remove_member';
      title: string;
      message: string;
      userID: number;
    }
  | {
      kind: 'remove_tech_lead';
      title: string;
      message: string;
      userID: number;
    };
